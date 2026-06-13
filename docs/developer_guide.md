# Developer Guide

A practical walkthrough for building native macOS applications using `go-bindings-macosplatform`. Code examples are drawn from [Orin](../app/orin/), a full-featured macOS virtual machine manager built on top of these bindings.

For API reference, generated package docs, and generator CLI usage, see the [root README](../README.md).

---

## Contents

- [Prerequisites](#prerequisites)
- [The Main Thread Rule](#the-main-thread-rule)
- [App Lifecycle](#app-lifecycle)
- [GC Root Pinning](#gc-root-pinning)
- [Creating a Window](#creating-a-window)
- [Building Menus](#building-menus)
- [Handling ObjC Blocks (Callbacks)](#handling-objc-blocks-callbacks)
- [Error Handling](#error-handling)
- [VM Lifecycle: An End-to-End Example](#vm-lifecycle-an-end-to-end-example)
- [Reactive State with Observables](#reactive-state-with-observables)
- [Persistent Configuration](#persistent-configuration)
- [Structured Logging](#structured-logging)

---

## Prerequisites

| Requirement | Version | Notes |
| --- | --- | --- |
| macOS | 13+ (Ventura) | Runtime requirement |
| Go | 1.26.2+ | Generics required for parameterised types |
| Xcode Command Line Tools | Latest | Needed only if re-running the generator |

Add the module:

```sh
go get github.com/deploymenttheory/go-bindings-macosplatform
```

All generated packages carry `//go:build darwin` — your application will only compile on macOS, which is expected.

---

## The Main Thread Rule

> **This is the most important concept in the entire SDK.** Violating it causes undefined behaviour and crashes that are extremely difficult to debug.

AppKit — and any other UI framework — **must only be called from the macOS main OS thread**. This is a hard requirement imposed by the macOS window server, not a convention.

`objc.RunOnMainThread` marshals a closure onto the main thread and blocks until it returns. Call it from any goroutine, including `main()`:

```go
//go:build darwin

package main

import (
    "context"

    orinapp "github.com/deploymenttheory/go-bindings-macosplatform/app/orin/app"
    orinlog "github.com/deploymenttheory/go-bindings-macosplatform/app/orin/log"
    "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
)

func main() {
    orinlog.Init()

    ctx := context.Background()
    // AppKit requires all UI calls on the main OS thread.
    objc.RunOnMainThread(func() {
        orinapp.Run(ctx)
    })
}
```

Under the hood `RunOnMainThread` calls `dispatch_sync_f(dispatch_get_main_queue(), ...)`. If you call it from the main thread itself, the closure runs inline without a dispatch round-trip.

**Non-UI work** (file I/O, networking, computation) is fine on any goroutine. Only calls into AppKit, CoreGraphics, Metal, and other UI frameworks need to be on the main thread.

---

## App Lifecycle

An AppKit application needs three things: an `NSApplication` shared instance, an activation policy (Regular = Dock icon + menu bar), and a run loop.

```go
//go:build darwin

package app

import (
    "context"

    rawappkit "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/appkit"
    "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
)

// Run is the application entry point. Must be called on the main thread.
func Run(ctx context.Context) {
    app := rawappkit.NSApplicationSharedApplication(ctx)
    // NSApplicationActivationPolicyRegular: show Dock icon + menu bar.
    app.SetActivationPolicy(ctx, rawappkit.NSApplicationActivationPolicyRegular)

    // Defer UI setup via dispatch_async so the Window Server connection is fully
    // established by [NSApp run]'s internal finishLaunching before any windows
    // or status items are created — equivalent to applicationDidFinishLaunching:.
    objc.DispatchAsyncMain(func() {
        setupUI(ctx, app)
    })

    // Blocks until the app terminates.
    app.Run(ctx)
}
```

`NSApplicationSharedApplication` creates the singleton `NSApplication` (equivalent to `[NSApplication sharedApplication]`).

`DispatchAsyncMain` schedules the closure to run on the next iteration of the main run loop, after AppKit completes its internal startup. This ensures the Window Server connection exists before you create any windows — equivalent to `applicationDidFinishLaunching:` in a delegate-based app.

---

## GC Root Pinning

Go's garbage collector scans the heap, but **Go stacks are not scanned during CGo calls**. This means any ObjC wrapper reachable only from the stack can be collected — and its finalizer run, releasing the ObjC object — while a CGo call is still using the underlying pointer.

The fix is to store long-lived wrappers in package-level variables so they are heap-reachable:

```go
// roots pins long-lived ObjC wrappers as GC roots. Go stacks are unscannable
// during CGo calls, so anything reachable only from the stack would be collected.
var roots struct {
    manager    *vm.VMManager
    settings   *ui.SettingsWindow
    library    *ui.LibraryWindow
    consoles   *ui.ConsoleWindows
    statusItem *rawappkit.NSStatusItem
    menuItems  *ui.AppMenuItems
    selection  *reactive.Observable[string]
}

// ... later, after all objects are created:
roots.manager    = manager
roots.settings   = settings
roots.library    = library
roots.consoles   = consoles
roots.statusItem = statusItem
roots.menuItems  = menuItems
roots.selection  = selection
```

Store all roots **before** starting goroutines that use them. The single struct makes it easy to audit what is pinned. Name it something obvious (`roots`, `appState`, `gcPins`) so the pattern is self-documenting.

---

## Creating a Window

### Using the opinionated library (recommended)

`opinionated/library/appkit` provides `NewWindow` with a `WindowConfig` struct that handles the full construction:

```go
import oappkit "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/appkit"

win := oappkit.NewWindow(ctx, oappkit.WindowConfig{
    Title:        "My App",
    Width:        820,
    Height:       480,
    Center:       true,         // call Center() after creation
    MinWidth:     680,
    MinHeight:    380,
    AutosaveName: "MyAppMain", // persists window position across launches
})

// Set the view that fills the window
oappkit.SetWindowContentView(ctx, win, myContentView)

// Show it
oappkit.ShowWindow(ctx, win)
```

`AutosaveName` calls `setFrameAutosaveName:` on the window, which causes AppKit to automatically save and restore the window's position and size in `NSUserDefaults`.

### Using raw bindings

The raw path gives you full control, at the cost of more ceremony:

```go
import (
    raw "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/appkit"
    ocf "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/corefoundation"
    of  "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/foundation"
)

w := raw.NewNSWindowWithContentRectStyleMaskBackingDefer(
    ctx,
    ocf.NewRect(0, 0, 820, 480),
    raw.NSWindowStyleMaskTitled |
        raw.NSWindowStyleMaskClosable |
        raw.NSWindowStyleMaskMiniaturizable |
        raw.NSWindowStyleMaskResizable,
    raw.NSBackingStoreBuffered,
    false,
)
w.SetTitle(ctx, of.StringFromGo("My App"))
w.SetMinSize(ctx, ocf.NewSize(680, 380))
w.Center(ctx)
w.MakeKeyAndOrderFront(ctx, nil)
```

---

## Building Menus

Menu bars are hierarchical: `NSMenuBar` → `NSMenu` (per top-level item) → `NSMenuItem` (per command).

The opinionated library's `appkit` package provides helpers that hide the boilerplate:

```go
import oappkit "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/appkit"

bar := oappkit.NewMenuBar(ctx)

// "File" menu
fileMenu := oappkit.NewMenu(ctx, "File")
oappkit.AddItem(ctx, fileMenu, oappkit.NewMenuItemWithAction(ctx, "New", "n", 0, func(_ unsafe.Pointer) {
    onNewVM()
}))
oappkit.AddSeparator(ctx, fileMenu)
oappkit.AddItem(ctx, fileMenu, oappkit.NewMenuItemWithAction(ctx, "Quit", "q", 0, func(_ unsafe.Pointer) {
    rawappkit.NSApplicationSharedApplication(ctx).Terminate(ctx, nil)
}))
oappkit.AddSubmenu(ctx, bar, "File", fileMenu)

app.SetMainMenu(ctx, bar)
```

### Keyboard modifier flags

The generated `NSEventModifierFlags` enum has sequential index values, not the bitmask values AppKit actually uses. Define the correct values explicitly:

```go
import raw "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/appkit"

// Keyboard modifier bitmasks — correct Apple values.
const (
    modCmd   = raw.NSEventModifierFlags(1 << 20) // ⌘
    modOpt   = raw.NSEventModifierFlags(1 << 19) // ⌥
    modCtrl  = raw.NSEventModifierFlags(1 << 18) // ⌃
    modShift = raw.NSEventModifierFlags(1 << 17) // ⇧
)

// Item with ⌘⇧S shortcut:
item := oappkit.NewMenuItemWithAction(ctx, "Save State", "s", modCmd|modShift, onSaveState)
```

### Enabling and disabling items dynamically

Keep `*NSMenuItem` references and call `SetEnabled` as state changes:

```go
type AppMenuItems struct {
    StartItem  *raw.NSMenuItem
    PauseItem  *raw.NSMenuItem
    ResumeItem *raw.NSMenuItem
    StopItem   *raw.NSMenuItem
}

func (m *AppMenuItems) UpdateForVM(ctx context.Context, inst *vm.VMInstance) {
    if inst == nil {
        m.StartItem.SetEnabled(ctx, false)
        m.PauseItem.SetEnabled(ctx, false)
        m.ResumeItem.SetEnabled(ctx, false)
        m.StopItem.SetEnabled(ctx, false)
        return
    }
    t := inst.State.Get().Transitions
    m.StartItem.SetEnabled(ctx, t.CanStart)
    m.PauseItem.SetEnabled(ctx, t.CanPause)
    m.ResumeItem.SetEnabled(ctx, t.CanResume)
    m.StopItem.SetEnabled(ctx, t.CanRequestStop)
}
```

---

## Handling ObjC Blocks (Callbacks)

Objective-C uses **blocks** — its equivalent of Go closures — as callback arguments to many APIs. A block has type `^(args) returnType` in ObjC; the bindings expose this as a Go `func`.

You pass a plain Go closure; the generated code wraps it in an ObjC block, passes it to the bridge call, and releases the block after the call returns:

```go
// NSArray.EnumerateObjectsUsingBlock takes a block: ^(id obj, NSUInteger idx, BOOL *stop)
arr.EnumerateObjectsUsing(ctx, func(obj unsafe.Pointer, idx uint64, stop unsafe.Pointer) {
    str := foundation.CastNSString(foundation.NSStringWithPtr(obj))
    fmt.Printf("[%d] %s\n", idx, of.StringToGo(str))
    // Set *stop = YES to terminate early
})
```

For block-based APIs that complete asynchronously (file operations, network calls, etc.), the opinionated library wraps them into straightforward `func(ctx) error` calls. See the [Opinionated Library](opinionated_library.md) guide for examples.

---

## Error Handling

### NSError → Go error

Methods with an `NSError **` out-parameter are generated to return a Go `error` as their last value:

```go
str, err := foundation.NSStringStringWithContentsOfURLEncodingError(ctx, url, encoding)
if err != nil {
    return fmt.Errorf("read file: %w", err)
}
```

The generated bridge captures the ObjC `NSError *`, converts it via `objc.NSErrorToError`, and returns it as a Go `error`. The ObjC object is released automatically.

### NSException → Go panic

Every generated bridge call wraps the ObjC method in `@try`/`@catch`. If ObjC throws `NSException`, it is caught and re-raised as a Go `panic`. Recover in the normal Go way:

```go
func safeObjectAt(arr *foundation.NSArray[*foundation.NSString], idx uint64) (result *foundation.NSString, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("ObjC exception: %v", r)
        }
    }()
    result = arr.ObjectAtIndex(ctx, idx).(*foundation.NSString)
    return
}
```

---

## VM Lifecycle: An End-to-End Example

The Orin app uses the Virtualization framework to run macOS and Linux VMs. This section walks through the key steps, showing both raw and opinionated APIs.

### Configure and validate a VM

```go
import (
    foundation "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/foundation"
    rawvz      "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/virtualization"
    virt       "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/virtualization"
)

func buildMacOSVM(ctx context.Context, cfg VMConfig) (*rawvz.VZVirtualMachine, error) {
    c := virt.NewVZVirtualMachineConfiguration(ctx)

    // CPU and memory
    c.SetCPUCount(ctx, cfg.CPUCount)
    c.SetMemorySize(ctx, uint64(cfg.MemoryGB)*1024*1024*1024)

    // macOS platform (hardware model + machine identifier + auxiliary storage)
    hwModel    := virt.NewMacHardwareModelFromBytes(ctx, cfg.HardwareModelData)
    machineID  := virt.NewMacMachineIdentifierFromBytes(ctx, cfg.MachineIDData)
    auxStorage := virt.LoadMacAuxiliaryStorage(ctx, cfg.AuxStoragePath)

    platform := rawvz.NewVZMacPlatformConfiguration(ctx)
    if err := virt.ApplyVZMacPlatformConfiguration(ctx, platform, virt.VZMacPlatformConfigurationSpec{
        HardwareModel:     hwModel,
        MachineIdentifier: machineID,
        AuxiliaryStorage:  auxStorage,
    }); err != nil {
        return nil, fmt.Errorf("mac platform: %w", err)
    }
    c.SetPlatform(ctx, &platform.VZPlatformConfiguration)

    // EFI boot loader
    bl, err := virt.NewEFIBootLoader(ctx, "")
    if err != nil {
        return nil, fmt.Errorf("EFI boot loader: %w", err)
    }
    c.SetBootLoader(ctx, &bl.VZBootLoader)

    // Display (1920×1200 at 80 ppi)
    display := virt.NewDisplayConfiguration(ctx, 1920, 1200, 80)
    gfxDev  := rawvz.NewVZMacGraphicsDeviceConfiguration(ctx)
    gfxDev.SetDisplays(ctx, foundation.NSArrayOf[objc.Object](ctx, display))
    c.SetGraphicsDevices(ctx, foundation.NSArrayOf[objc.Object](ctx, gfxDev))

    // Disk
    if cfg.DiskPath != "" {
        attach, err := virt.NewDiskImageAttachment(ctx, cfg.DiskPath, false)
        if err != nil {
            return nil, fmt.Errorf("disk attachment: %w", err)
        }
        diskDev := rawvz.NewVZVirtioBlockDeviceConfigurationWithAttachment(
            ctx, &attach.VZStorageDeviceAttachment)
        c.SetStorageDevices(ctx, foundation.NSArrayOf[objc.Object](ctx, diskDev))
    }

    // Validate before use
    if ok, err := c.ValidateWithError(ctx); !ok || err != nil {
        if err != nil {
            return nil, fmt.Errorf("VM config invalid: %w", err)
        }
        return nil, fmt.Errorf("VM config invalid")
    }

    return rawvz.NewVZVirtualMachineWithConfigurationQueue(ctx, c, nil), nil
}
```

### Lifecycle commands (opinionated library)

The opinionated `virtualization` package wraps the callback-based start/pause/resume/stop APIs into blocking, context-aware functions:

```go
// Start the VM — blocks until started or error.
if err := virt.Start(ctx, machine); err != nil {
    return fmt.Errorf("VM start: %w", err)
}

// Pause
if err := virt.Pause(ctx, machine); err != nil {
    return fmt.Errorf("VM pause: %w", err)
}

// Resume
if err := virt.Resume(ctx, machine); err != nil {
    return fmt.Errorf("VM resume: %w", err)
}

// Request graceful shutdown
if err := virt.RequestStop(ctx, machine); err != nil {
    return fmt.Errorf("VM stop: %w", err)
}
```

### Polling VM state

`VZVirtualMachine.State` can only be read on the main thread:

```go
go func() {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-stopCh:
            return
        case <-ticker.C:
            var state rawvz.VZVirtualMachineState
            objc.RunOnMainThread(func() {
                state = machine.State(ctx)
            })
            // update observable, trigger UI refresh, etc.
        }
    }
}()
```

---

## Reactive State with Observables

`opinionated/library/reactive` provides a generic observable value that can be subscribed to by multiple consumers. It is safe to read and write from any goroutine.

```go
import reactive "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/reactive"

// Create an observable with an initial value.
selection := reactive.New("")  // Observable[string]

// Read the current value.
id := selection.Get()

// Write a new value — all subscribers are notified synchronously.
selection.Set("vm-uuid-1234")

// Subscribe — the closure is called immediately with the current value,
// then again each time Set is called.
selection.Subscribe(func(id string) {
    // Always dispatch UI work back to the main thread.
    objc.RunOnMainThread(func() {
        menuItems.UpdateForVM(ctx, findVM(id))
    })
})
```

The Orin app uses observables to connect VM state changes to UI updates without any shared-mutex coordination:

```go
// VM state poller (goroutine) → observable → UI subscriber (main thread)
inst.State.Subscribe(func(s VMStateSnapshot) {
    objc.RunOnMainThread(func() {
        rebuildAll()       // update list view, toolbar, menu items
        updateStatusIcon() // update system status bar
    })
})
```

---

## Persistent Configuration

Use JSON serialisation with atomic file writes to persist application state:

```go
type VMConfig struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    CPUCount   uint   `json:"cpu_count"`
    MemoryGB   uint   `json:"memory_gb"`
    DiskPath   string `json:"disk_path,omitempty"`
}

func SaveConfigs(cfgs []VMConfig) error {
    path, err := configFilePath()
    if err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return fmt.Errorf("create config dir: %w", err)
    }
    data, err := json.MarshalIndent(cfgs, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal config: %w", err)
    }
    // Write to a temp file, then rename — atomic on the same filesystem.
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0o644); err != nil {
        return fmt.Errorf("write config: %w", err)
    }
    return os.Rename(tmp, path)
}

func configFilePath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, "Library", "Application Support", "MyApp", "vms.json"), nil
}
```

The `os.WriteFile(tmp) + os.Rename` pattern is atomic on APFS: a crash during `WriteFile` leaves the old config intact; only a successful `Rename` commits the new one.

---

## Structured Logging

The Orin app uses [zerolog](https://github.com/rs/zerolog) to write structured JSON to `~/Library/Logs/orin/orin.log` while also printing human-readable output to stderr:

```go
import "github.com/rs/zerolog"

func initLogger() zerolog.Logger {
    logPath := filepath.Join(homeDir(), "Library", "Logs", "MyApp", "myapp.log")
    _ = os.MkdirAll(filepath.Dir(logPath), 0o755)

    f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)

    console := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
    multi   := zerolog.MultiLevelWriter(console, f)

    level := zerolog.InfoLevel
    if os.Getenv("MYAPP_DEBUG") == "1" {
        level = zerolog.DebugLevel
    }

    return zerolog.New(multi).Level(level).With().Timestamp().Logger()
}

// Usage
log.Info().
    Str("vm_id", inst.Config.ID).
    Str("name", inst.Config.Name).
    Str("state", string(phase)).
    Msg("VM state changed")
```

The structured fields make log aggregation and filtering straightforward without losing human-readable output during development.
