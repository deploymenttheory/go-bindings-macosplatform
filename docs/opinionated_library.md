# The Idiomatic Binding API

> **History.** This page used to describe a separate `opinionated/library/` tree of
> ergonomic helpers layered on top of raw ObjC-shaped bindings. That split is gone.
> The ergonomics are now **built into the single consumable binding API** at
> `bindings/frameworks/<name>` and `bindings/libraries/<name>`. The old
> `opinionated/library/`, `opinionated/idiomatic/`, and `opinionated/custom/` trees
> no longer exist (`opinionated/` now holds only hand-written `opinionated/tools/`).
> This page explains what "idiomatic" buys you, with before/after comparisons against
> the ObjC-shaped **raw substrate** that now lives internally.

There is **one** consumable binding API, and it is the idiomatic one:

- `bindings/frameworks/<name>` — the ObjC frameworks (Foundation, AppKit, Virtualization, …). 252 packages.
- `bindings/libraries/<name>` — the Apple C libraries (libproc, xpc, dispatch, EndpointSecurity, …). 16 packages.

The **raw** purego/CGo bindings still exist, but they are now internal plumbing under
`bindings/internal/raw/frameworks/<name>` and `bindings/internal/raw/libraries/<name>`.
Go's internal-package rule makes them unreachable from outside `bindings/`, so as a
consumer you never import them. The idiomatic layer is built on top of that substrate
(and uses it as a parity oracle during generation), but you only ever name the
idiomatic package.

---

## Contents

- [What "Idiomatic" Means](#what-idiomatic-means)
- [Import Conventions](#import-conventions)
- [Before/After Comparisons](#beforeafter-comparisons)
  - [1. Strings](#1-strings)
  - [2. Window Creation](#2-window-creation)
  - [3. Async Callbacks](#3-async-callbacks)
  - [4. Typed Collections](#4-typed-collections)
  - [5. Configuration Builders](#5-configuration-builders)
  - [6. Geometry](#6-geometry)
  - [7. VM Configuration and Lifecycle](#7-vm-configuration-and-lifecycle)
- [Naming](#naming)
- [Where the Hand-Written Helpers Went](#where-the-hand-written-helpers-went)

---

## What "Idiomatic" Means

The raw substrate reflects Objective-C idioms directly: `*NSString` everywhere,
`completionHandler:` blocks, untyped `NSArray`, one setter per property, exact ObjC
selector names. It is complete and correct, but it adds ceremony to every call site.

The idiomatic layer reshapes that surface into Go:

- **Go types at the boundary** — methods take and return `string`, `[]byte`, `int`,
  `bool`, `time.Time`, and typed Go slices instead of ObjC wrapper pointers.
- **Go errors** — an `NSError **` out-parameter becomes a trailing `error` return; an
  `OSStatus` becomes an `error`.
- **`func(ctx) error` async** — a `completionHandler:` block becomes a blocking call
  that takes a `context.Context` and returns when the operation completes or `ctx` is
  cancelled.
- **Chainable `With*` setters** — each settable property has a `WithX(v) *T` method
  that sets and returns the receiver, so configuration reads as a fluent chain.
- **Go-shaped names** — `NSString` → `foundation.String`, `NSView` → `appkit.View`
  (see [Naming](#naming)).
- **Automatic memory management** — wrappers retain on creation and release via a GC
  finalizer; you just let them go out of scope.
- **Automatic main-thread dispatch** — methods, setters, and constructors of an
  `@MainActor`-isolated class (AppKit and everything that inherits it) are wrapped in
  `purego.Main` for you, so UI calls land on the main thread automatically.

None of the raw idioms were bugs — they are accurate representations of how the macOS
SDK works. The idiomatic layer bridges the gap between "correct binding" and
"idiomatic Go" without hiding what happens underneath.

---

## Import Conventions

Import the idiomatic package directly — no `raw` alias, because you never import raw:

```go
import (
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/appkit"
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/corefoundation"
)
```

When you also drop to the runtime (for a custom ObjC subclass, NSXPC, or dispatch),
import the public runtime and convert with the `obj` package:

```go
import (
    rt  "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/obj"
)
```

---

## Before/After Comparisons

Each "raw" block below shows the ObjC-shaped surface (the shape the internal raw
substrate exposes); the "idiomatic" block shows the consumable API you actually call.

### 1. Strings

Converting between Go `string` and ObjC `*NSString` is the most frequent Foundation
operation.

**Raw shape:** methods traffic in `*NSString`; to read one you call `UTF8String()`
(returning an `unsafe.Pointer`) and copy it into Go; to build one you go through a
runtime helper. Every hop needs a `KeepAlive` to stop the GC releasing the object
mid-call.

**Idiomatic:** methods take and return Go `string` directly. There is almost nothing
to convert:

```go
info := foundation.NSProcessInfoProcessInfo()
name := info.ProcessName()          // string, not *NSString
```

When you genuinely need a `*String` wrapper (to pass where a Foundation object is
expected), build it from a Go string and read it back with `.String()`:

```go
s := foundation.NewStringWithString("Hello, macOS") // *foundation.String
go := s.String()                                     // back to a Go string
```

At the runtime level, `purego.NSString(goStr)` and `purego.GoString(id)` do the raw
`objc.ID` conversion.

---

### 2. Window Creation

Creating a standard titled, closable, resizable window.

**Raw shape:** call the constructor, then a separate `SetTitle:`, `SetMinSize:`,
`SetFrameAutosaveName:`, `Center`, `MakeKeyAndOrderFront:` — every one an ObjC-shaped
call taking `*NSString`/`NSSize`, and every one needing a manual main-thread dispatch.

**Idiomatic:** chainable `With*` setters, Go values, and automatic main-thread
dispatch:

```go
w := appkit.NewWindowWithContentRectStyleMaskBackingDefer(
    corefoundation.CGRect{Size: corefoundation.CGSize{Width: 820, Height: 480}},
    appkit.WindowStyleMaskTitled|appkit.WindowStyleMaskClosable|appkit.WindowStyleMaskResizable,
    appkit.BackingStoreBuffered,
    false,
).
    WithTitle("VM Library").
    WithMinSize(corefoundation.CGSize{Width: 680, Height: 380})

w.Center()
w.MakeKeyAndOrderFront(nil)
```

`WithTitle` takes a Go `string`; every setter runs on the main thread automatically
because `Window` inherits AppKit's `@MainActor` isolation.

---

### 3. Async Callbacks

Many Foundation APIs are callback-based: they accept a completion block that fires
asynchronously.

**Raw shape** (`NSFileManager`): the method is
`UnmountVolumeAtURLOptionsCompletionHandler(url, mask, completionHandler func(...))`.
To use it like a normal Go function you create a channel, pass a closure that sends on
it, and `select` on both the channel and `ctx.Done()` — at every async call site.

**Idiomatic:** the completion handler is folded into a blocking, context-aware call:

```go
// Blocks until the operation completes or ctx is cancelled.
err := fileManager.UnmountVolumeAtURLOptions(ctx, url, mask)
```

The idiomatic method takes `context.Context` as its first argument and returns the
`NSError` (if any) as a Go `error`. Context cancellation is handled uniformly — you
never write the `select` block.

---

### 4. Typed Collections

ObjC collections (`NSArray`, `NSMutableArray`) are untyped at the C boundary.

**Raw shape:** a getter returns an `NSArray`; to read elements you call
`ObjectAtIndex:` and cast each `unsafe.Pointer` result to the concrete type.

**Idiomatic:** getters return typed Go slices, and setters accept them:

```go
// Returns []string — no casting, no index arithmetic.
urls := fileManager.DirectoryURLs(foundation.DocumentDirectory,
    foundation.UserDomainMask)
for _, u := range urls {
    // u is a Go string
}
```

Element conversion (`NSString` → `string`, `NSURL` → `string`, wrapper → wrapper) is
done for you inside the accessor.

---

### 5. Configuration Builders

Configuring an object with many properties — a session config, a date formatter, a VM
config — is one setter per property in the raw shape.

**Idiomatic:** the `With*` setters chain, so configuration is a single fluent
expression and you set only what you care about:

```go
c := virtualization.NewVirtualMachineConfiguration().
    WithCPUCount(4).
    WithMemorySize(4 * 1024 * 1024 * 1024)
```

Each `WithX` returns the receiver, so there is no intermediate variable and no
per-setter main-thread bookkeeping.

---

### 6. Geometry

CoreGraphics value types (`CGSize`, `CGPoint`, `CGRect`) are plain Go structs in the
idiomatic `corefoundation` package — construct them with struct literals:

```go
rect := corefoundation.CGRect{
    Origin: corefoundation.CGPoint{X: 0, Y: 0},
    Size:   corefoundation.CGSize{Width: 820, Height: 480},
}
```

They are ordinary value types (no ObjC object, no memory management), so they pass by
value into any method that takes a `CGRect`/`CGSize`/`CGPoint`.

---

### 7. VM Configuration and Lifecycle

The Virtualization framework is queue-confined (not `@MainActor`), so its calls are
**not** main-thread wrapped — you run them on the VM's own dispatch queue (see
`opinionated/tools/grandcentraldispatch/serialqueue`).

**Configuration** chains the same way as any other object:

```go
c := virtualization.NewVirtualMachineConfiguration().
    WithCPUCount(cfg.CPUCount).
    WithMemorySize(uint64(cfg.MemoryGB) * 1024 * 1024 * 1024)
```

**Lifecycle** commands (`start:`, `pause:`, `resume:`) are completion-handler based in
ObjC; idiomatically they are blocking, context-aware calls:

```go
if err := machine.Start(ctx); err != nil {
    return fmt.Errorf("VM start: %w", err)
}
```

---

## Naming

The idiomatic layer trims a class-name prefix **only when every class in a framework
shares one** (≥2 uppercase chars):

| ObjC class | Idiomatic Go type |
|---|---|
| `NSString` | `foundation.String` |
| `NSProcessInfo` | `foundation.ProcessInfo` |
| `NSView` | `appkit.View` |
| `OSSystemExtensionManager` | `systemextensions.SystemExtensionManager` |

When a framework's classes don't share a prefix, the full name is kept (NetworkExtension
mixes `NE…`/`NW…`, so it stays `networkextension.NEFilterNewFlowVerdict`).

A class method keeps its full ObjC name as a package-level function but returns the
idiomatic type — e.g. `foundation.NSProcessInfoProcessInfo() *foundation.ProcessInfo`,
then `.ProcessIdentifier() int`. Instance selectors become methods
(`objectAtIndex:` → `ObjectAtIndex`). Two reliable lookups when you're unsure of a
trimmed name:

1. Each wrapper's doc comment names the class it wraps —
   `grep -rn "wrapper over the Objective-C class NSView" bindings/frameworks/appkit`.
2. Every generated package has a `doc.go` index listing its types.

The full naming contract is [`naming.md`](naming.md).

---

## Where the Hand-Written Helpers Went

The old `opinionated/library/` helpers (declarative `NewWindow`, `Apply*` spec types,
typed-slice wrappers, async wrappers, the `reactive` observable) are gone — the
idiomatic layer provides the same ergonomics natively, as shown above. The only
hand-written packages that remain are under `opinionated/tools/`, for capabilities that
still don't map to "send a selector to an object":

| Package | Purpose |
|---|---|
| `opinionated/tools/grandcentraldispatch/mainthread` | Own and service the main thread; `Do(fn)`, `IsMain()`, run-loop pumping |
| `opinionated/tools/grandcentraldispatch/serialqueue` | A dedicated serial dispatch queue for queue-confined frameworks (e.g. Virtualization) |
| `opinionated/tools/keychain` | Go-shaped `SecItem*` helpers (certificates, identities, passwords) |
| `opinionated/tools/oslog` | Emit to the unified logging system (needs a small CGo shim) |

See [`opinionated/tools/README.md`](../opinionated/tools/README.md) for details.
