# Developer Guide

A practical walkthrough for building native macOS applications using `go-bindings-macosplatform`. All examples call the single consumable binding API — the **idiomatic** layer at `bindings/frameworks/<name>` and `bindings/libraries/<name>` — dropping to `bindings/runtime/...` only for things that have no typed wrapper. See the runnable [`examples/`](../examples/) (`keychain`, `warden`) for worked programs.

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

**The idiomatic layer handles this for you.** Every method, setter, and constructor of an `@MainActor`-isolated class — AppKit and everything that inherits its isolation — is wrapped in `purego.Main` automatically, so a plain `w.WithTitle("…")` already runs on the main thread. You only manage the main thread yourself when you (a) drop to the runtime and send selectors by hand, or (b) need to own the run loop.

`purego.Main` marshals a closure onto the main thread and blocks until it returns; call it from any goroutine. When called from the main thread it runs the closure inline (a `dispatch_sync` to the main queue from the main thread would deadlock):

```go
//go:build darwin

package main

import (
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/appkit"
    purego "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

func main() {
    // The idiomatic AppKit calls below already dispatch to the main thread, but the
    // process must own and service the main thread for that dispatch to be serviced.
    purego.Main(func() {
        run()
    })
}

func run() {
    app := appkit.SharedApplication()
    app.SetActivationPolicy(appkit.ApplicationActivationPolicyRegular)
    // … build UI …
    app.Run()
}
```

To own and pump the main run loop directly (drive it yourself rather than calling `[NSApp run]`), use the `opinionated/tools/grandcentraldispatch/mainthread` helper.

**Non-UI work** (file I/O, networking, computation) is fine on any goroutine. Only calls into AppKit, CoreGraphics, Metal, and other UI frameworks need to be on the main thread.

---

## App Lifecycle

An AppKit application needs three things: an `NSApplication` shared instance, an activation policy (Regular = Dock icon + menu bar), and a run loop.

```go
//go:build darwin

package app

import (
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/appkit"
)

// Run is the application entry point. Must be called on the main thread.
func Run() {
    app := appkit.SharedApplication()
    // ApplicationActivationPolicyRegular: show Dock icon + menu bar.
    app.SetActivationPolicy(appkit.ApplicationActivationPolicyRegular)

    // Build the UI (windows, menus, status items) before starting the run loop.
    setupUI(app)

    // Blocks until the app terminates.
    app.Run()
}
```

`appkit.SharedApplication()` creates the singleton `NSApplication` (equivalent to `[NSApplication sharedApplication]`). It is a class method, so the ObjC selector `+sharedApplication` becomes a package-level function.

These idiomatic AppKit calls take no `context.Context` and are dispatched to the main thread automatically. If you need UI setup to run *after* AppKit finishes its internal startup (equivalent to `applicationDidFinishLaunching:`), schedule it onto the main queue with the `opinionated/tools/grandcentraldispatch/mainthread` helper so it fires on the next run-loop iteration.

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
    statusItem *appkit.StatusItem
    menuItems  *ui.AppMenuItems
}

// ... later, after all objects are created:
roots.manager    = manager
roots.settings   = settings
roots.library    = library
roots.consoles   = consoles
roots.statusItem = statusItem
roots.menuItems  = menuItems
```

Store all roots **before** starting goroutines that use them. The single struct makes it easy to audit what is pinned. Name it something obvious (`roots`, `appState`, `gcPins`) so the pattern is self-documenting.

---

## Creating a Window

Construct the window, then configure it with chainable `With*` setters. Each setter runs on the main thread automatically (`Window` inherits AppKit's `@MainActor` isolation), takes Go values, and returns the receiver so the calls chain:

```go
import (
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/appkit"
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/corefoundation"
)

w := appkit.NewWindowWithContentRectStyleMaskBackingDefer(
    corefoundation.CGRect{Size: corefoundation.CGSize{Width: 820, Height: 480}},
    appkit.WindowStyleMaskTitled|
        appkit.WindowStyleMaskClosable|
        appkit.WindowStyleMaskMiniaturizable|
        appkit.WindowStyleMaskResizable,
    appkit.BackingStoreBuffered,
    false,
).
    WithTitle("My App").
    WithMinSize(corefoundation.CGSize{Width: 680, Height: 380}).
    WithContentView(myContentView) // ViewProvider — any *appkit.View subclass

w.Center()
w.MakeKeyAndOrderFront(nil)
```

`WithFrameAutosaveName` calls `setFrameAutosaveName:` on the window, which causes AppKit to automatically save and restore the window's position and size in `NSUserDefaults`.

---

## Building Menus

Menu bars are hierarchical: the main menu (`NSMenu`) → a `NSMenu` per top-level item → `NSMenuItem` per command. Build the tree with the idiomatic `appkit` types and attach it with the chainable `WithMainMenu` setter:

```go
import "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/appkit"

bar      := appkit.NewMenuWithTitle("MainMenu")
fileMenu := appkit.NewMenuWithTitle("File")

// The top-level item and the command items are created through the runtime
// (see note below); once you hold a *appkit.MenuItem you add it idiomatically.
fileItem := newCommandItem("File", nil) // runtime-backed helper, returns *appkit.MenuItem
bar.AddItem(fileItem)
bar.SetSubmenuForItem(fileMenu, fileItem)

// A separator, by contrast, is a real idiomatic class method:
fileMenu.AddItem(appkit.SeparatorItem())

app.WithMainMenu(bar)
```

**Creating a command item and wiring it to a Go function** is the part the idiomatic layer can't do on its own. It exposes no `initWithTitle:action:keyEquivalent:` constructor, and a menu item's action is an ObjC target + selector — the idiomatic wrappers configure *existing* objects, they can't define a new target class. So you build the command item and its target through the runtime (`bindings/runtime/purego`): alloc the `NSMenuItem`, register a delegate with `rt.NewDelegate`, and point the item's `setTarget:`/`setAction:` at a selector that delegate implements. Lift the resulting id into the idiomatic API with `obj.Wrap(id)` when you need to. See [`examples/warden`](../examples/warden/) for the runtime-subclass pattern and the [examples adoption guide](../examples/README.md) for crossing the idiomatic↔runtime boundary.

Once you hold a `*appkit.MenuItem`, its title, key equivalent, and enabled state are chainable idiomatic setters — `WithTitle`, `WithKeyEquivalent`, `WithKeyEquivalentModifierMask`, `WithEnabled`.

### Keyboard modifier flags

The idiomatic `EventModifierFlags` enum carries the correct Apple bitmask values, so use the constants directly — no manual redefinition needed:

```go
// EventModifierFlagCommand == 1<<20 (⌘), EventModifierFlagShift == 1<<17 (⇧), etc.
mask := appkit.EventModifierFlagCommand | appkit.EventModifierFlagShift // ⌘⇧
```

### Enabling and disabling items dynamically

Keep `*appkit.MenuItem` references and use the `WithEnabled` setter as state changes:

```go
type AppMenuItems struct {
    StartItem  *appkit.MenuItem
    PauseItem  *appkit.MenuItem
    ResumeItem *appkit.MenuItem
    StopItem   *appkit.MenuItem
}

func (m *AppMenuItems) UpdateForVM(inst *vm.VMInstance) {
    if inst == nil {
        m.StartItem.WithEnabled(false)
        m.PauseItem.WithEnabled(false)
        m.ResumeItem.WithEnabled(false)
        m.StopItem.WithEnabled(false)
        return
    }
    t := inst.Transitions()
    m.StartItem.WithEnabled(t.CanStart)
    m.PauseItem.WithEnabled(t.CanPause)
    m.ResumeItem.WithEnabled(t.CanResume)
    m.StopItem.WithEnabled(t.CanRequestStop)
}
```

---

## Handling ObjC Blocks (Callbacks)

Objective-C uses **blocks** — its equivalent of Go closures — as callback arguments to many APIs. A block has type `^(args) returnType` in ObjC; the bindings expose this as a Go `func`.

You pass a plain Go closure; the generated code wraps it in an ObjC block, passes it to the bridge call, and releases the block after the call returns:

```go
// NSArray.enumerateObjectsUsingBlock: takes a block: ^(id obj, NSUInteger idx, BOOL *stop)
arr.EnumerateObjectsUsing(func(item obj.Object, idx int, stop *bool) {
    if s := foundation.StringFromID(obj.ID(item)); s != nil {
        fmt.Printf("[%d] %s\n", idx, s.String())
    }
    // Set *stop = true to terminate early.
})
```

For block-based APIs that complete asynchronously (file operations, network calls, etc.), the idiomatic layer folds the completion block into a blocking `func(ctx) error` call natively — no wrapper package needed. See the [idiomatic API guide](opinionated_library.md) for examples.

---

## Error Handling

### NSError → Go error

Methods with an `NSError **` out-parameter are generated to return a Go `error` as their last value:

```go
str, err := foundation.StringWithContentsOfURLEncoding(url, encoding)
if err != nil {
    return fmt.Errorf("read file: %w", err)
}
```

The generated code captures the ObjC `NSError *`, converts it via the runtime (`purego.NSErrorToError`, decoded into a structured `errkit`/`objcerrors` error), and returns it as a Go `error`. The ObjC object is released automatically.

### NSException → Go panic

Every generated bridge call wraps the ObjC method in `@try`/`@catch`. If ObjC throws `NSException`, it is caught and re-raised as a Go `panic`. Recover in the normal Go way:

```go
func safeObjectAt(arr *foundation.Array, idx int) (result *foundation.String, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("ObjC exception: %v", r)
        }
    }()
    result = foundation.StringFromID(obj.ID(arr.ObjectAtIndex(idx)))
    return
}
```

---

## VM Lifecycle: An End-to-End Example

The Virtualization framework runs macOS and Linux VMs. It is **queue-confined**, not `@MainActor`: a `VZVirtualMachine` and its configuration must be used on the serial dispatch queue you create it on (see `opinionated/tools/grandcentraldispatch/serialqueue`), so these calls are *not* main-thread wrapped.

### Configure and validate a VM

Configuration chains with the idiomatic `With*` setters. The device setters take typed providers (or a variadic list of them), not untyped `NSArray`s:

```go
import "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/virtualization"

func buildMacOSVM(cfg VMConfig) (*virtualization.VirtualMachine, error) {
    c := virtualization.NewVirtualMachineConfiguration().
        WithCPUCount(cfg.CPUCount).
        WithMemorySize(uint64(cfg.MemoryGB) * 1024 * 1024 * 1024)

    // Attach the platform, boot loader, and devices with the chainable setters:
    // WithPlatform, WithBootLoader, WithStorageDevices, WithGraphicsDevices, … .
    // Build the provider objects (VZMacPlatformConfiguration, VZEFIBootLoader,
    // VZVirtioBlockDeviceConfiguration, …) with their own idiomatic constructors
    // in the virtualization package.
    c.WithBootLoader(bootLoader).
        WithPlatform(platform).
        WithStorageDevices(diskDevice).
        WithGraphicsDevices(graphicsDevice)

    // Validate before use — returns a Go error (nil when the config is valid).
    if err := c.Validate(); err != nil {
        return nil, fmt.Errorf("VM config invalid: %w", err)
    }

    return virtualization.NewVirtualMachineWithConfigurationQueue(c, queue), nil
}
```

### Lifecycle commands

The callback-based start/pause/resume APIs are folded into blocking, context-aware methods; `RequestStop` and `State` are synchronous:

```go
if err := machine.Start(ctx); err != nil { // blocks until started or ctx cancelled
    return fmt.Errorf("VM start: %w", err)
}
if err := machine.Pause(ctx); err != nil {
    return fmt.Errorf("VM pause: %w", err)
}
if err := machine.Resume(ctx); err != nil {
    return fmt.Errorf("VM resume: %w", err)
}
if err := machine.RequestStop(); err != nil { // request graceful shutdown
    return fmt.Errorf("VM stop: %w", err)
}
```

### Polling VM state

Because a `VZVirtualMachine` must be touched only on the serial queue it was created on, read `State()` from that queue:

```go
import "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/tools/grandcentraldispatch/serialqueue"

go func() {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-stopCh:
            return
        case <-ticker.C:
            var state virtualization.VirtualMachineState
            vmQueue.Do(func() { // vmQueue is the serialqueue the VM was created on
                state = machine.State()
            })
            // trigger a UI refresh with the new state, etc.
        }
    }
}()
```

---

## Reactive State (App-Level Pattern)

The SDK does not ship an observable/reactive helper — that is application concern, not a binding. A tiny plain-Go observable is all you need to connect background state changes (e.g. a VM state poller) to UI updates. The one SDK-specific rule: **dispatch the UI work back to the main thread** with `purego.Main`, since AppKit is `@MainActor`.

```go
import purego "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"

// A minimal observable — a value plus subscribers, safe from any goroutine.
type Observable[T any] struct {
    mu   sync.Mutex
    val  T
    subs []func(T)
}

func (o *Observable[T]) Set(v T) {
    o.mu.Lock()
    o.val = v
    subs := append([]func(T){}, o.subs...)
    o.mu.Unlock()
    for _, fn := range subs {
        fn(v)
    }
}

func (o *Observable[T]) Subscribe(fn func(T)) {
    o.mu.Lock()
    o.subs = append(o.subs, fn)
    o.mu.Unlock()
}
```

Wire a background poller to the UI without any shared-mutex coordination — the subscriber hops to the main thread itself:

```go
// VM state poller (goroutine) → observable → UI subscriber (main thread)
state.Subscribe(func(s VMStateSnapshot) {
    purego.Main(func() {
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

A typical macOS app uses [zerolog](https://github.com/rs/zerolog) to write structured JSON to `~/Library/Logs/<app>/<app>.log` while also printing human-readable output to stderr:

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
