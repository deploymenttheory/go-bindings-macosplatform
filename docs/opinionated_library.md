# Opinionated Library

The `opinionated/library/` tree is a collection of ergonomic helpers that sit on top of the raw `frameworks/` packages. This page explains why it exists, what it contains, and shows concrete side-by-side comparisons between using the raw bindings and using the opinionated layer.

---

## Contents

- [Why It Exists](#why-it-exists)
- [What It Contains](#what-it-contains)
- [Import Conventions](#import-conventions)
- [Side-by-Side Comparisons](#side-by-side-comparisons)
  - [1. String Conversion](#1-string-conversion)
  - [2. Window Creation](#2-window-creation)
  - [3. Async Callbacks](#3-async-callbacks)
  - [4. Typed Collections](#4-typed-collections)
  - [5. Configuration Builders (Spec Types)](#5-configuration-builders-spec-types)
  - [6. Geometry Helpers](#6-geometry-helpers)
  - [7. VM Configuration](#7-vm-configuration)
- [Generated Helper Categories](#generated-helper-categories)
- [Hand-Crafted Helpers by Domain](#hand-crafted-helpers-by-domain)
- [Extending the Layer](#extending-the-layer)

---

## Why It Exists

The raw `frameworks/` packages are **complete and correct**. Every Objective-C class, method, enum, and struct is faithfully represented with full CGo bridges. You can build real applications using only the raw bindings.

However, the raw bindings reflect Objective-C idioms directly:

- **Callbacks via blocks** — async operations take a closure argument (`completionHandler:`). To use them from Go you need to create a channel, pass a closure that sends on it, and then select on both the channel and `ctx.Done()`.
- **`NSArray` everywhere** — ObjC collections are untyped at the C level. Getting typed elements requires calling `ObjectAtIndex:` and casting the result.
- **Setter-per-property configuration** — configuring an `NSURLSessionConfiguration` means calling 20+ individual setter methods, all of which must be dispatched on the right thread.
- **Two-step string conversion** — going between Go `string` and `*NSString` requires calling through `objc.GoStringToNSString` / `objc.NSStringToGoString` via the `objc` internal package.
- **Long generated names** — constructor names like `NewNSWindowWithContentRectStyleMaskBackingDefer` are faithful to the ObjC selector but noisy in Go call sites.

None of these are bugs. They are accurate representations of how the macOS SDK works. But they add ceremony to every call site.

The opinionated layer's purpose is to bridge the gap between "correct CGo binding" and "idiomatic Go code" — providing Go-shaped wrappers without hiding what is happening underneath.

**The raw bindings are never modified.** The opinionated layer sits on top; everything compiles to the same CGo calls.

---

## What It Contains

```text
opinionated/library/
├── foundation/          # hand-crafted: string helpers, NSData, Progress, FileHandle
│   ├── string.go
│   ├── data.go
│   ├── progress.go
│   ├── foundation_async_generated.go    ← generated
│   ├── foundation_slices_generated.go   ← generated
│   └── foundation_specs_generated.go    ← generated
├── appkit/              # hand-crafted: window, menu, alert, file picker, status item
│   ├── window.go
│   ├── menu.go
│   ├── alert.go
│   ├── filepicker.go
│   ├── statusitem.go
│   ├── appkit_async_generated.go        ← generated
│   ├── appkit_slices_generated.go       ← generated
│   └── appkit_specs_generated.go        ← generated
├── virtualization/      # hand-crafted: VM lifecycle, platform config, storage, network
│   ├── machine.go
│   ├── macos.go
│   ├── network.go
│   ├── storage.go
│   └── virtualization_specs_generated.go ← generated
├── corefoundation/      # hand-crafted: CGSize/CGPoint/CGRect constructors
│   └── geometry.go
├── bsd/                 # hand-crafted: EtherAddr ↔ string helpers
│   └── etheraddr.go
├── reactive/            # hand-crafted: generic observable value
│   └── observable.go
└── <other frameworks>/  # generated only (*_async_generated.go, *_slices_generated.go,
                         #              *_specs_generated.go)
```

There are three categories of generated helper, one category of hand-crafted helper, and one utility package (`reactive`):

| Category | File suffix | What it does |
| --- | --- | --- |
| Async wrappers | `*_async_generated.go` | Wraps callback-based APIs into `func(ctx) error` calls |
| Typed slices | `*_slices_generated.go` | Returns `[]*ConcreteType` instead of raw `NSArray` |
| Spec types | `*_specs_generated.go` | Struct + `Apply*` function to configure objects declaratively |
| Hand-crafted | (any non-generated file) | Domain-specific helpers too nuanced to generate |

---

## Import Conventions

Import raw bindings with the `raw` alias; import opinionated packages with a short domain alias:

```go
import (
    raw    "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/appkit"
    oappkit "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/appkit"
    of     "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/foundation"
)
```

Never import from `opinionated/` inside a `frameworks/` package, and never import from `frameworks/` inside `opinionated/` (only from `raw` aliases as shown above). The dependency flows one way: `opinionated` → `frameworks`.

---

## Side-by-Side Comparisons

### 1. String Conversion

Converting between Go `string` and ObjC `*NSString` is one of the most frequent operations when using any Foundation API.

**Raw:**

```go
import (
    raw  "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/foundation"
    "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
)

// Go string → *NSString
ns := raw.NewNSString(objc.GoStringToNSString("Hello, macOS"))

// *NSString → Go string
s := objc.NSStringToGoString(ns.Ptr())
objc.KeepAlive(ns)
```

**Opinionated:**

```go
import of "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/foundation"

// Go string → *NSString
ns := of.StringFromGo("Hello, macOS")

// *NSString → Go string (returns "" if ns is nil)
s := of.StringToGo(ns)
```

The opinionated wrappers handle the `KeepAlive` call internally and guard against nil receivers — both common sources of bugs at raw call sites.

---

### 2. Window Creation

Creating a standard titled, closable, resizable window.

**Raw:**

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
w.SetTitle(ctx, of.StringFromGo("VM Library"))
w.SetMinSize(ctx, ocf.NewSize(680, 380))
w.SetFrameAutosaveName(ctx, of.StringFromGo("OrinLibrary"))
w.Center(ctx)
w.MakeKeyAndOrderFront(ctx, nil)
```

**Opinionated:**

```go
import oappkit "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/appkit"

w := oappkit.NewWindow(ctx, oappkit.WindowConfig{
    Title:        "VM Library",
    Width:        820,
    Height:       480,
    Center:       true,
    MinWidth:     680,
    MinHeight:    380,
    AutosaveName: "OrinLibrary",
})
oappkit.ShowWindow(ctx, w)
```

`NewWindow` applies sensible defaults (standard style mask, buffered backing store) and wires up `AutosaveName` automatically. You opt into only the properties you care about.

---

### 3. Async Callbacks

Many Foundation APIs are callback-based: they accept a completion block that fires asynchronously. Using them from Go requires boilerplate channel/select code to make them behave like normal Go functions.

**Raw (example: unmount a volume):**

```go
// You write this at every async call site:
ch := make(chan error, 1)
fileManager.UnmountVolumeAtURLOptionsCompletionHandler(ctx, url, mask, func(err error) {
    ch <- err
})
var err error
select {
case err = <-ch:
case <-ctx.Done():
    err = ctx.Err()
}
return err
```

**Opinionated** (`foundation_async_generated.go`, auto-generated):

```go
import of "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/foundation"

// Blocks until the operation completes or ctx is cancelled.
err := of.UnmountVolumeAtURLOptions(ctx, fileManager, url, mask)
```

The generated async wrapper is produced for every method whose name ends in `CompletionHandler` (or similar). The pattern is always:

```go
func OperationName(ctx context.Context, recv *raw.Type, args...) error {
    ch := make(chan error, 1)
    recv.OperationNameCompletionHandler(ctx, args..., func(err error) { ch <- err })
    select {
    case err := <-ch:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

Context cancellation is handled uniformly — you never need to write the select block yourself.

---

### 4. Typed Collections

ObjC collections (`NSArray`, `NSMutableArray`, etc.) are untyped at the C boundary. Getting elements requires calling `ObjectAtIndex:` and casting the `unsafe.Pointer` result to the concrete type.

**Raw (iterating over date formatter era symbols):**

```go
symbols := formatter.EraSymbols(ctx) // *NSArray[objc.Object]
count := int(symbols.Count(ctx))
result := make([]string, count)
for i := 0; i < count; i++ {
    elem := raw.CastNSString(symbols.ObjectAtIndex(ctx, uint64(i)))
    result[i] = objc.NSStringToGoString(elem.Ptr())
}
```

**Opinionated** (`foundation_slices_generated.go`, auto-generated):

```go
import of "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/foundation"

// Returns a typed Go slice — no casting, no index arithmetic.
symbols := of.EraSymbolsList(ctx, formatter) // []*raw.NSString
result := make([]string, len(symbols))
for i, ns := range symbols {
    result[i] = of.StringToGo(ns)
}
```

Generated slice helpers come in pairs: `XyzList(ctx, recv)` to read (returns a typed slice), and `SetXyzList(ctx, recv, items)` to write (accepts a typed slice, converts to `NSArray` internally).

---

### 5. Configuration Builders (Spec Types)

Configuring objects with many properties — `NSURLSessionConfiguration`, `VZVirtualMachineConfiguration`, `NSDateFormatter`, etc. — normally requires calling a separate setter for each property.

**Raw (configuring an NSURLSessionConfiguration):**

```go
config := raw.NewNSURLSessionDefaultSessionConfiguration(ctx)
config.SetRequestCachePolicy(ctx, raw.NSURLRequestReloadIgnoringLocalCacheData)
config.SetTimeoutIntervalForRequest(ctx, 30.0)
config.SetTimeoutIntervalForResource(ctx, 300.0)
config.SetAllowsCellularAccess(ctx, true)
config.SetAllowsExpensiveNetworkAccess(ctx, false)
config.SetHTTPShouldUsePipelining(ctx, true)
config.SetHTTPShouldSetCookies(ctx, true)
config.SetWaitsForConnectivity(ctx, true)
// ... 15 more setters
```

**Opinionated** (`foundation_specs_generated.go`, auto-generated):

```go
import of "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/foundation"

config := raw.NewNSURLSessionDefaultSessionConfiguration(ctx)
of.ApplyNSURLSessionConfiguration(ctx, config, of.NSURLSessionConfigurationSpec{
    RequestCachePolicy:           raw.NSURLRequestReloadIgnoringLocalCacheData,
    TimeoutIntervalForRequest:    30.0,
    TimeoutIntervalForResource:   300.0,
    AllowsCellularAccess:         true,
    AllowsExpensiveNetworkAccess: false,
    HTTPShouldUsePipelining:      true,
    HTTPShouldSetCookies:         true,
    WaitsForConnectivity:         true,
})
```

The `Apply*` function **only calls setters for non-zero fields**. Leave a field at its zero value and the corresponding setter is never called — the existing default is preserved. This also avoids redundant ObjC round-trips.

---

### 6. Geometry Helpers

CoreGraphics value types (`CGSize`, `CGPoint`, `CGRect`) require nested struct literals in the raw bindings.

**Raw:**

```go
import cf "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/corefoundation"

rect := cf.CGRect{
    Origin: cf.CGPoint{X: 0, Y: 0},
    Size:   cf.CGSize{Width: 820, Height: 480},
}
```

**Opinionated:**

```go
import ocf "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/corefoundation"

rect  := ocf.NewRect(0, 0, 820, 480)
size  := ocf.NewSize(820, 480)
point := ocf.NewPoint(10, 20)

// Geometry predicate
inside := ocf.RectContainsPoint(rect, mouseX, mouseY)
```

---

### 7. VM Configuration

Configuring a `VZVirtualMachineConfiguration` requires building multiple device arrays and passing them as `NSArray`.

**Raw (storage + network + keyboard, stripped for brevity):**

```go
// Disk
attach, _ := rawvz.NewVZDiskImageStorageDeviceAttachmentWithURLReadOnlyError(
    ctx, diskURL, false)
diskDev := rawvz.NewVZVirtioBlockDeviceConfigurationWithAttachment(
    ctx, &attach.VZStorageDeviceAttachment)
c.SetStorageDevices(ctx, foundation.NSArrayOf[objc.Object](ctx, diskDev))

// Network
netDev := rawvz.NewVZVirtioNetworkDeviceConfiguration(ctx)
natAtt := rawvz.NewVZNATNetworkDeviceAttachment(ctx)
netDev.SetAttachment(ctx, &natAtt.VZNetworkDeviceAttachment)
c.SetNetworkDevices(ctx, foundation.NSArrayOf[objc.Object](ctx, netDev))

// Keyboard
kbd := rawvz.NewVZUSBKeyboardConfiguration(ctx)
c.SetKeyboards(ctx, foundation.NSArrayOf[objc.Object](ctx, kbd))
```

**Opinionated** (using a spec that handles all device arrays):

```go
import virt "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/virtualization"

virt.ApplyVZVirtualMachineConfiguration(ctx, c, virt.VZVirtualMachineConfigurationSpec{
    StorageDevices: []rawvz.VZStorageDeviceConfiguration{diskDev},
    NetworkDevices: []rawvz.VZNetworkDeviceConfiguration{netDev},
    Keyboards:      []rawvz.VZKeyboardConfiguration{kbd},
})
```

The `Apply` function converts Go slices to `NSArray`, wraps items as `objc.Object`, and calls each setter exactly once.

---

## Generated Helper Categories

### Async wrappers (`*_async_generated.go`)

Generated for every method whose Go name ends in `CompletionHandler`, `WithReply`, or similar. All follow the same channel-select pattern:

```go
// Auto-generated signature pattern:
func OperationName(ctx context.Context, recv *raw.Type, args...) (result, error) {
    ch := make(chan result, 1)
    recv.OperationNameCompletionHandler(ctx, args..., func(r result, err error) {
        // pack both into ch
    })
    select {
    case v := <-ch:
        return v.result, v.err
    case <-ctx.Done():
        return zero, ctx.Err()
    }
}
```

### Typed slices (`*_slices_generated.go`)

Generated for every getter/setter that returns or accepts an `NSArray`:

```go
// Read: NSArray → typed Go slice
func XyzList(ctx context.Context, recv *raw.Type) []*raw.ElementType

// Write: typed Go slice → NSArray
func SetXyzList(ctx context.Context, recv *raw.Type, items []*raw.ElementType)
```

### Spec types (`*_specs_generated.go`)

Generated for every class with multiple settable properties:

```go
type TypeNameSpec struct {
    FieldA TypeA  // zero value = skip setter
    FieldB TypeB
    // ...
}

func ApplyTypeName(ctx context.Context, recv *raw.TypeName, spec TypeNameSpec) error
```

---

## Hand-Crafted Helpers by Domain

### `opinionated/library/foundation`

| Helper | Purpose |
| --- | --- |
| `StringFromGo(s string) *NSString` | Go string → `*NSString` |
| `StringToGo(ns *NSString) string` | `*NSString` → Go string, nil-safe |
| `NSDataToBytes(d *NSData) []byte` | Efficient byte copy using `unsafe.Slice` |
| `BytesToNSData(b []byte) *NSData` | `[]byte` → `*NSData` |
| `GetProgress(p *NSProgress) ProgressSnapshot` | Read all progress fields in one call |

### `opinionated/library/appkit`

| Helper | Purpose |
| --- | --- |
| `NewWindow(ctx, WindowConfig) *NSWindow` | Declarative window creation |
| `ShowWindow / HideWindow / CloseWindow` | Window visibility lifecycle |
| `NewMenu / AddItem / AddSeparator / AddSubmenu` | Menu bar construction |
| `NewMenuItemWithAction(ctx, title, key, mods, fn)` | Menu item with Go callback |
| `NewStatusItem(ctx, StatusItemConfig) *NSStatusItem` | Status bar item |
| `PickFile / PickFiles / PickDirectory / SaveFile` | File picker helpers |
| `ShowAlert(ctx, AlertConfig)` | Modal alert without boilerplate |
| `NewHSplitView / NewVSplitView` | Split view construction |
| `SetWindowContentView` | Set content view on a window |

### `opinionated/library/virtualization`

| Helper | Purpose |
| --- | --- |
| `Start / Pause / Resume / RequestStop` | Blocking VM lifecycle commands with context |
| `NewEFIBootLoader(ctx, nvramPath)` | EFI boot loader with optional NVRAM |
| `NewLinuxBootLoader / NewLinuxBootLoaderWithInitrd` | Linux kernel boot loader |
| `NewMacHardwareModelFromBytes` | Deserialise hardware model from stored bytes |
| `NewMacMachineIdentifierFromBytes` | Deserialise machine ID from stored bytes |
| `LoadMacAuxiliaryStorage / CreateMacAuxiliaryStorage` | Auxiliary storage management |
| `NewDiskImageAttachment(ctx, path, readOnly)` | Disk image storage attachment |
| `NewDisplayConfiguration(ctx, width, height, ppi)` | Short form display config |
| `GetVMTransitions(ctx, machine)` | Read all allowed transitions in one call |

### `opinionated/library/corefoundation`

| Helper | Purpose |
| --- | --- |
| `NewSize(w, h float64) CGSize` | CGSize constructor |
| `NewPoint(x, y float64) CGPoint` | CGPoint constructor |
| `NewRect(x, y, w, h float64) CGRect` | CGRect constructor |
| `RectContainsPoint(rect, x, y)` | Point-in-rect predicate |

### `opinionated/library/reactive`

| Helper | Purpose |
| --- | --- |
| `New[T](initial T) *Observable[T]` | Create a new observable |
| `(o) Get() T` | Read current value |
| `(o) Set(v T)` | Write new value, notify subscribers |
| `(o) Subscribe(fn func(T))` | Register a subscriber |

---

## Extending the Layer

**Hand-crafted files are never deleted by the generator.** Only files ending in `_generated.go` are regenerated. You can add new helpers alongside generated files without risk.

Naming conventions:
- Hand-crafted files: descriptive name (`window.go`, `machine.go`, `geometry.go`)
- Generated files: `<framework>_async_generated.go`, `<framework>_slices_generated.go`, `<framework>_specs_generated.go`

Cross-package imports within `opinionated/library/` use the full module path:
```go
import oappkit "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/appkit"
```

Raw framework imports within opinionated files use the `raw` alias:
```go
import raw "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/appkit"
```
