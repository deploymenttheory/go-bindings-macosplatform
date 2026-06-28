# go-bindings-macosplatform

Type-safe Go bindings for native macOS framework APIs, generated directly from the installed version of the Xcode SDK on macOS deterministically.

This project provides three things:

- A **code generator** that introspects macOS SDK headers via Clang and produces idiomatic Go packages — ObjC frameworks are bound through [purego](https://github.com/ebitengine/purego) (no CGo, no Xcode needed to build your app), and Apple C libraries are bound through CGo bridges.
- The **generated bindings** themselves — ready-to-import Go packages covering 250 ObjC frameworks (`bindings/frameworks/`) and 11 Apple C libraries (`bindings/libraries/`) discovered in the macOS SDK.
- An **opinionated idiomatic layer** (`opinionated/idiomatic/`) built on top of the raw bindings — fluent, Go-shaped wrappers where constructors bundle `alloc`+`init`, properties become chainable `With*` setters, async completion handlers become `func(ctx) error`, `NSArray` getters become typed Go slices, and C functions get prefix-stripped Go names. Subclasses **embed their base** (inheriting its methods through Go promotion); an abstract base's setters accept a **sealed provider interface** so only real members of the hierarchy type-check; abstract bases emit no meaningless constructor; multi-value methods use **named returns**; and each package's `doc.go` carries a type index so `go doc` reads like a manual. Calls that Apple isolates to the **main thread** (`@MainActor` — AppKit and everything that inherits from it, like `MKMapView`) are wrapped in `purego.Main` **automatically**, so UI code is correct without the caller remembering to dispatch. The layer is **hermetic** — it never imports the raw bindings, dispatching straight through the runtime.

> **Platform:** macOS only (`darwin`). All generated code carries a `//go:build darwin` constraint.

---

## Documentation

| Guide | Description |
| --- | --- |
| [Developer Guide](docs/developer_guide.md) | Build macOS apps with the SDK — app lifecycle, windows, menus, VM management, blocks, and more |
| [Opinionated Library](docs/opinionated_library.md) | Why the opinionated layers exist, their benefits, and raw vs. opinionated side-by-side comparisons |
| [Extraction Workflow](docs/extraction_workflow.md) | How Clang AST scanning produces `.gometa.json` files and how those drive Go code generation |
| [Naming Standard](docs/naming.md) | The naming contract for generator code and generated identifiers |
| [Metadata Overrides](docs/metadata_overrides.md) | Declarative per-framework metadata corrections applied at load time |

---

## Quick Start

Add the module to your project:

```sh
go get github.com/deploymenttheory/go-bindings-macosplatform
```

Import whichever frameworks you need and use the generated types directly. ObjC framework packages are pure Go — the framework dylib is loaded at runtime via `dlopen`, so no CGo toolchain is required to compile against them:

```go
package main

import (
    "fmt"

    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
)

func main() {
    // ObjC class methods become package-level functions: +[NSProcessInfo processInfo]
    info := foundation.NSProcessInfoProcessInfo()
    fmt.Println(info.ProcessIdentifier(), info.ProcessorCount())

    // Go strings convert at the boundary: +[NSString stringWithUTF8String:]
    ns := foundation.NSStringStringWithUTF8String("Hello, macOS")
    fmt.Println(ns.Length()) // 12
}
```

The idiomatic layer trades raw fidelity for fluency. Each `With*` setter returns the
receiver, so configuration reads as a single expression, and setters accept *sealed*
provider interfaces so only a real member of a class hierarchy type-checks:

```go
import vz "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/virtualization"

config := vz.NewVirtualMachineConfiguration().
    WithBootLoader(vz.NewLinuxBootLoaderWithKernelURL("/var/vm/vmlinuz")).
    WithCPUCount(2).
    WithMemorySize(2 << 30)

// WithBootLoader accepts vz.BootLoaderProvider — only VZBootLoader subclasses
// satisfy it, so passing a non-boot-loader is a compile error, not a runtime panic.
```

---

## Contents

- [Documentation](#documentation)
- [How It Works](#how-it-works)
- [Prerequisites](#prerequisites)
- [Using the Generated Bindings](#using-the-generated-bindings)
  - [Import Paths](#import-paths)
  - [Object Model](#object-model)
  - [C Functions](#c-functions)
  - [Memory Management](#memory-management)
  - [ObjC Blocks](#objc-blocks)
  - [Error Handling](#error-handling)
  - [Main Thread Dispatch](#main-thread-dispatch)
- [Generating Bindings](#generating-bindings)
  - [CLI Reference](#cli-reference)
  - [Pipeline Phases](#pipeline-phases)
- [Architecture](#architecture)
  - [Repository Layout](#repository-layout)
  - [Two Emission Pipelines](#two-emission-pipelines)
  - [Runtime Layer](#runtime-layer)
  - [Generated Package Structure](#generated-package-structure)
  - [Generated Code Annotations](#generated-code-annotations)
  - [Metadata QA](#metadata-qa)
- [Framework Coverage](#framework-coverage)
- [Known Limitations](#known-limitations)
- [Contributing](#contributing)
- [License](#license)

---

## How It Works

```mermaid
flowchart LR
    SDK["macOS SDK\n(Xcode headers)"]
    Clang["clang\n-ast-dump=json"]
    Meta[".gometa.json\ncached metadata"]
    Gen["cmd/generate"]
    Pure["bindings/frameworks/…\n(purego — no CGo)"]
    CGo["bindings/libraries/…\n(CGo bridges)"]
    Idio["opinionated/idiomatic/…\n(fluent wrappers)"]
    App["Your Go app"]

    SDK --> Clang --> Meta --> Gen
    Gen --> Pure --> App
    Gen --> CGo --> App
    Pure --> Idio --> App
```

The generator uses Clang to dump the full AST of each framework header, extracts metadata (classes, protocols, enums, structs, free functions, extern constants, block types, availability windows, deprecation messages, and doc comments) into JSON, and then emits Go source files.

ObjC frameworks are emitted as **pure Go** packages: classes resolve via `objc_getClass`, methods dispatch through `objc.Send`, and the framework dylib is `dlopen`ed lazily at package init. Apple C libraries (EndpointSecurity, xpc, dispatch, …) are emitted as **CGo** packages with generated `.h`/`.m` bridge files.

The result is a set of Go packages where every Objective-C class becomes a Go struct, selectors become Go methods, inheritance is modelled via struct embedding, and C functions become exported Go functions — all with automatic memory management through Go finalizers.

---

## Prerequisites

| Requirement | Version | Notes |
| --- | --- | --- |
| macOS | 13+ (Ventura) | Required for framework dylibs and the Objective-C runtime |
| Go | 1.26.2+ | Generics required for parameterised types (e.g. `NSArray[T]`) |
| Xcode Command Line Tools | Latest | Required only for `scan` (Clang/SDK headers) and for building apps that import the CGo `bindings/libraries/` packages |

Install Xcode Command Line Tools if you haven't already:

```sh
xcode-select --install
```

Verify Clang is available via `xcrun`:

```sh
xcrun clang --version
```

**External dependencies:** `github.com/ebitengine/purego` is the foundational dependency — it provides `dlopen`, `objc_msgSend` dispatch, and block support without CGo. `go.opentelemetry.io/otel` traces every CGo C-library call. The remaining entries in `go.mod` (Sentry, MCP, gRPC, …) serve the in-repo example applications and tooling, not the bindings themselves.

---

## Using the Generated Bindings

### Import Paths

Each macOS framework maps to a Go package under `bindings/frameworks/`, and Apple C libraries map to `bindings/libraries/`:

```go
import (
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/vmnet"
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/libraries/xpc"
)
```

The idiomatic layer mirrors the same package names under `opinionated/idiomatic/framework/`
(C-library wrappers live under `opinionated/idiomatic/libraries/`):

```go
import (
    fluent "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
)
```

Available packages mirror the frameworks in this repository — see [Framework Coverage](#framework-coverage) for the categories covered.

### Object Model

Every Objective-C class is a Go struct. Superclass relationships are modelled using struct embedding, giving you inherited methods directly on the subtype:

```go
// NSMutableString embeds NSString which embeds NSObject
type NSMutableString struct {
    NSString
}

// Methods from NSObject and NSString are promoted automatically
var s *foundation.NSMutableString
_ = s.Length() // NSString method, available via embedding
```

**Factory functions:** ObjC class methods (e.g. `+[NSString stringWithUTF8String:]`) are generated as package-level functions named after their class and selector:

```go
// +[NSString stringWithUTF8String:] → NSStringStringWithUTF8String
ns := foundation.NSStringStringWithUTF8String("hello")

// +[NSDate date] → NSDateDate
now := foundation.NSDateDate()
```

**Wrapping a raw object id:** If you hold a raw `objc.ID` (from purego, a block callback, or an XPC/dispatch pointer), wrap it with the generated `<Class>FromID` constructor. This registers a Go finalizer so the underlying ObjC object is released when the Go wrapper is collected:

```go
str := foundation.NSStringFromID(id) // nil if id == 0
```

> `FromID` registers a *releasing* finalizer but does **not** retain — retain first (`purego.Retain(id)`) if you don't own a +1 reference.

**Generic containers:** Generic Objective-C containers use Go generics:

```go
var arr *foundation.NSArray[*foundation.NSString]
count := arr.Count()
```

**Protocols** are emitted as plain Go interfaces (`foundation.NSCopying`, …) for typing and duck-typed acceptance.

### C Functions

Free C functions are exported with Go-style names. Functions whose C names are already exported identifiers keep them byte-for-byte (`CFArrayCreate`, `SecItemAdd`); snake_case names become PascalCase, with the original symbol recorded in a doc comment:

```go
// C function: vmnet_start_interface
iface := vmnet.VmnetStartInterface(desc, queue, handler)
```

The idiomatic layer additionally strips the framework prefix (types stay in the raw package):

```go
import (
    "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/vmnet"
    idvmnet "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/vmnet"
)

var status vmnet.Vmnet_return_t
config := idvmnet.NetworkConfigurationCreate(vmnet.VMNET_SHARED_MODE, &status)
```

**Symbol availability:** some headers declare functions that have no dylib export (header-inline helpers, or symbols removed in newer macOS releases). Every generated package with C functions exposes a guard — calling a wrapper whose symbol failed to bind panics on a nil function variable, so probe first when in doubt:

```go
if !vmnet.SymbolAvailable("vmnet_start_interface") {
    return errors.New("vmnet unavailable on this system")
}
```

### Memory Management

Object-returning methods retain the result (+1) and `FromID` constructors register a Go finalizer; when the GC collects the Go wrapper, the underlying Objective-C object is released. In most cases you do not need to manage memory manually.

```mermaid
sequenceDiagram
    participant Go as Go GC
    participant W as Go Wrapper
    participant ObjC as ObjC Runtime

    Note over W,ObjC: method returns an object
    W->>ObjC: [obj retain]
    W->>Go: purego.Track → runtime.SetFinalizer(wrapper, release)

    Note over Go: wrapper unreachable
    Go->>W: finalizer fires
    W->>ObjC: [obj release]
```

In the CGo `bindings/libraries/` packages, Objective-C ARC is **disabled** in all bridge files (`-fno-objc-arc`); reference counting is likewise driven from the Go side so the two runtimes never fight over ownership.

### ObjC Blocks

APIs that accept block callbacks take plain Go closures. The generated code wraps the closure in a real ObjC block object via `objc.NewBlock`, converts the callback arguments (object ids are retained and wrapped, `char*` becomes `string`), and releases the block after the call — escaping completion handlers stay alive because the callee copies them:

```go
done := make(chan vmnet.Vmnet_return_t, 1)

iface := vmnet.VmnetStartInterface(desc, queue,
    func(status vmnet.Vmnet_return_t, params *foundation.NSObject) {
        done <- status
    })
```

Block signatures whose components cannot cross purego's callback ABI (struct-by-value arguments, protocol interfaces, float returns) degrade to an `objc.Block` parameter instead — construct the block yourself with `objc.NewBlock` for those. Every degradation is recorded in the committed diagnostics baseline.

The CGo C-library packages use generated block trampolines (`bindings/runtime/blocks`) for the same effect — you still just pass a Go closure.

### Error Handling

**ObjC errors as Go errors:** Methods with an `NSError **` out-parameter return a Go `error` as their last value, carrying the structured NSError domain, code, description, and failure reason:

```go
path := foundation.NSStringStringWithUTF8String("/etc/hosts")
data, err := foundation.NSDataDataWithContentsOfFileOptionsError(path, 0)
if err != nil {
    log.Fatal(err) // structured: domain, code, description, failure reason
}
```

In the idiomatic layer, `BOOL`-returning error methods collapse to a plain `error`, and C functions with `CFErrorRef *` out-parameters return `(result, error)` with the CFError converted via its toll-free NSError bridge.

**ObjC exceptions:** the CGo `bindings/libraries/` packages wrap every call in `@try`/`@catch` and re-raise exceptions as Go panics (recorded on the active OTel span). The purego `bindings/frameworks/` packages do **not** intercept ObjC exceptions — an uncaught `NSException` terminates the process, as it would in an ObjC program.

### Main Thread Dispatch

All AppKit (and generally all UI-related) calls **must run on the macOS main thread**. Failure to do so causes undefined behaviour and crashes.

**The idiomatic layer handles this for you.** Apple isolates UI APIs to Swift's `@MainActor`; that isolation is harvested from the Swift symbol graph and propagated down the class hierarchy, so every method, `With*` setter, and constructor of a main-thread-bound class — `NSWindow`, `NSView`, and everything that inherits from them, including `MKMapView`, `PDFView`, `SCNView` in other frameworks — is wrapped in `purego.Main` automatically. When you already hold the main thread the wrapper runs inline (no dispatch), so there is no cost on the hot path and no deadlock. Queue-based frameworks (Virtualization, Core Data) are left untouched, since they require a consistent serial queue rather than the main thread. You still need a running main run loop (`NSApplication.Run`, a CFRunLoop, or `dispatch_main`) for the dispatched work to execute.

The **raw** bindings (`bindings/frameworks/`) do **not** dispatch automatically — there, the caller is responsible. The standard pattern for a UI app is to lock the main goroutine to the main OS thread, hand it to the AppKit run loop, and dispatch UI work from other goroutines via the `mainthread` helper package (pure Go, GCD main queue under the hood):

```go
import "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/mainthread"

func init() { runtime.LockOSThread() }

func main() {
    go func() {
        mainthread.Do(func() {
            // UI work — runs on the main thread, Do blocks until it returns
        })
    }()

    appkit.NSApplicationSharedApplication().Run() // run loop drains the main queue
}
```

`mainthread.Do` runs inline when already on the main thread (avoiding the dispatch-to-self deadlock), and `mainthread.IsMain()` reports which thread you're on. Note that dispatched work only executes while the main thread services its run loop — `NSApplication.Run`, a CFRunLoop, or `dispatch_main`. Lower-level GCD access is available via the `bindings/libraries/dispatch` package.

---

## Generating Bindings

The generator lives in `cmd/generate` and uses a subcommand interface. Run it with `go run`:

```sh
go run ./cmd/generate/ <subcommand> [flags]
```

### CLI Reference

| Subcommand | Description |
| --- | --- |
| `scan` | Invoke Clang on SDK headers and write `.gometa.json` metadata (requires Xcode) |
| `bindings` | Re-emit bindings from committed metadata: purego ObjC frameworks + CGo C libraries (no Clang needed) |
| `idiomatic` | Re-emit the `opinionated/idiomatic/` layer |
| `class-hierarchy` | Derive the canonical ObjC class hierarchy → `metadata/objcclasshierarchy/` |
| `all` | Run scan (optional) + bindings in sequence |
| `validate` | Structural integrity checks over committed metadata (runs in CI) |
| `diff` | Semantic API diff between two metadata trees (for reviewable SDK bumps) |
| `list` | List all frameworks the installed SDK exposes and exit |

**Common flags:**

| Flag | Description |
| --- | --- |
| `--framework <name\|all\|A,B,...>` | Framework(s) to operate on. Use `all` for every SDK framework. |
| `--frameworks-out <path>` | Output directory for ObjC framework packages (default: `./bindings/frameworks`) |
| `--libraries-out <path>` | Output directory for C library packages (default: `./bindings/libraries`) |
| `--diagnostics-baseline <file>` | Fail if regeneration produces a type degradation not in the committed baseline (CI ratchet) |
| `--diagnostics <file>` | Rewrite the diagnostics baseline (deliberate, reviewed change) |
| `-v` | Verbose output (type degradations, cycle breaks) |

**Common workflows:**

```sh
# Re-emit all bindings from committed metadata (fast — no Clang, no Xcode required)
go run ./cmd/generate/ bindings

# Re-emit and enforce the committed diagnostics baseline (what CI runs)
go run ./cmd/generate/ bindings --diagnostics-baseline metadata/diagnostics-baseline.json

# Re-emit the idiomatic layer (all frameworks, or one)
go run ./cmd/generate/ idiomatic
go run ./cmd/generate/ idiomatic --framework Virtualization

# Scan a single framework and write .gometa.json to metadata/ (requires Xcode)
go run ./cmd/generate/ scan --framework Foundation

# Scan + generate everything for one framework, or the whole SDK
go run ./cmd/generate/ all --framework Foundation
go run ./cmd/generate/ all --framework all

# Validate committed metadata / diff two metadata trees
go run ./cmd/generate/ validate
go run ./cmd/generate/ diff --old /tmp/old/metadata --new ./metadata

# List all frameworks the current SDK exposes
go run ./cmd/generate/ list
```

> **Tip:** The repository ships with pre-built `.gometa.json` files for all discovered frameworks in `metadata/`. This means `go run ./cmd/generate/ bindings` is the fast everyday path — no Xcode or Clang invocation needed.

### Pipeline Phases

```mermaid
flowchart TD
    subgraph Phase1["Phase 1 — Scan"]
        H["Framework header\n(e.g. Foundation.h)"]
        C["xcrun clang\n-ast-dump=json"]
        EX["Extract metadata\n(classes, enums, structs,\navailability, attrs…)"]
        HP["Raw header parsing\n(API_AVAILABLE, doc comments)"]
        JSON[".gometa.json\n(per framework × arch)"]
        H --> C --> EX --> JSON
        H --> HP --> JSON
    end

    subgraph Phase2["Phase 2 — Load"]
        J2[".gometa.json files\n+ overrides.json fixups"]
        REG["Registry\n(cross-framework type index,\ncanonical class hierarchy)"]
        J2 --> REG
    end

    subgraph Phase3["Phase 3 — Emit raw"]
        PURE["purego frameworks\n(objc.Send, dlopen,\nblock adapters)"]
        CGO["CGo C libraries\n(.h/.m bridges,\nOTel tracing)"]
    end

    subgraph Phase4["Phase 4 — Emit idiomatic"]
        IDIO["opinionated/idiomatic/\n(constructors, With* chains,\nasync wrappers, typed slices,\nC-function wrappers)"]
    end

    Phase1 --> Phase2 --> Phase3 --> Phase4
```

**Phase 1 — Scan:** For each framework, invokes `xcrun clang -x objective-c -ast-dump=json` against the umbrella header. The AST is walked to extract classes, protocols, enums, structs, free functions, extern constants, and block types. Availability annotations (`API_AVAILABLE`, `API_DEPRECATED`, …) and doc comments are parsed from the raw SDK header source using the AST-reported line numbers as anchors — required because Apple clang 21+ (Xcode 26.x) no longer embeds platform/version data inside `AvailabilityAttr` JSON nodes. Apple C libraries that live under `{SDK}/usr/include/` rather than `System/Library/Frameworks/` (EndpointSecurity, xpc, dispatch, …) are registered in `metadata/clibraries.json`.

**Phase 2 — Load:** All `.gometa.json` files are read and merged into a `Registry` that indexes every known class, its owning framework, and whether it uses generics. Canonical ownership is determined by the "fewest non-zero methods wins" heuristic; declarative per-framework fixups in `metadata/<kind>/<name>/overrides.json` are applied so committed metadata stays pure scanned output. When metadata exists for multiple architectures, `arm64` is preferred.

**Phase 3 — Emit raw bindings:** ObjC frameworks are emitted as purego packages in topological dependency order; mutual-import cycles are detected via DFS and broken by degrading the cross-framework reference to `objc.ID`. C libraries are emitted as CGo packages with generated bridge files. Every type degradation is collected and checked against `metadata/diagnostics-baseline.json` — new degradations fail CI until deliberately accepted.

**Phase 4 — Emit idiomatic:** The `idiomatic` subcommand regenerates `opinionated/idiomatic/` — per-class fluent wrappers (subclasses embedding their base), sealed provider interfaces for abstract base classes, CFError-converting function wrappers, generic C-function wrappers, and a `doc.go` type index per package. The emitter is a compiler-style pipeline: a resolution pass turns scanned metadata into a pure-data intermediate representation (the `view` package), and a render pass turns that into Go source through `text/template` files only — no Go syntax is assembled by string concatenation, and imports are computed from the resolved types rather than scanned from the output. Hand-crafted packages under `opinionated/library/` and `opinionated/custom/`, and any hand-authored `example_test.go`, are never touched by the generator.

---

## Architecture

### Repository Layout

```text
go-bindings-macosplatform/
├── cmd/
│   ├── generate/          # Main CLI (scan, bindings, idiomatic, class-hierarchy, all, list, validate, diff)
│   ├── inspect/           # Debug utility to inspect .gometa.json files
│   └── genacceptance/     # Regenerates the acceptance test corpus
├── bindings/
│   ├── frameworks/        # Generated purego packages for ObjC frameworks (250)
│   │   ├── foundation/
│   │   ├── appkit/
│   │   └── …
│   ├── libraries/         # Generated CGo packages for Apple C libraries (11)
│   │   ├── endpointsecurity/
│   │   ├── xpc/
│   │   ├── dispatch/
│   │   └── …
│   └── runtime/           # Public runtime — imported by generated code AND by consumers
│       ├── purego/        # purego runtime: Track/Retain/Release, GoString, NSErrorToError + ObjC dispatch re-exports (+ objcerrors/)
│       ├── cgo/           # CGo runtime: retain/release, RunOnMainThread, exceptions
│       ├── blocks/        # CGo block trampoline runtime
│       └── callbacks/     # CGo method/callback trampoline runtime
├── opinionated/
│   ├── idiomatic/         # Fully generated fluent layer
│   │   ├── framework/     #   one package per ObjC framework
│   │   ├── libraries/     #   one package per Apple C library
│   │   └── obj/ rt/ errkit/ internal/objref/   # generated runtime support packages
│   ├── library/           # Hand-crafted quality-of-life helpers (never regenerated)
│   └── custom/            # Custom hand-crafted packages
├── internal/
│   ├── macosplatformmetadata/  # Canonical scanned-SDK model + .gometa.json I/O (shared by scanner + both pipelines + QA)
│   ├── scanner/           # Clang AST dump, metadata extraction, raw header parsing, C library registry
│   ├── codegen/
│   │   ├── frameworks/    # purego pipeline: loader, typemap, naming, emitters (frameworks + idiomatic)
│   │   └── libraries/     # CGo pipeline: loader, typemap, naming, emitters (C libraries)
│   ├── validate/ metadiff/ diagnostics/ overrides/  # Metadata QA machinery
│   └── swift/             # Swift-only framework support (.swiftinterface parser + stub emit)
├── example/ weave/        # In-repo example applications built on the bindings
├── acceptance/            # Acceptance tests (sampled live calls + curated regression anchors)
├── docs/                  # Guides, naming standard, overrides format
└── metadata/              # Committed .gometa.json cache (per framework × arch)
    ├── frameworks/        # ObjC framework metadata (+ optional overrides.json)
    ├── libraries/         # Apple C library metadata (+ optional overrides.json)
    ├── clibraries.json    # C library registry (name → link_lib/header)
    └── diagnostics-baseline.json  # Known type degradations (CI ratchet)
```

### Two Emission Pipelines

| | ObjC frameworks (`bindings/frameworks/`) | Apple C libraries (`bindings/libraries/`) |
| --- | --- | --- |
| Bridge | purego (`objc.Send`, `dlopen` at init) | CGo (`.h`/`.m` bridge files, `-fno-objc-arc`) |
| Build requirements | Pure Go — no Xcode/Clang | CGo — Clang at build time |
| Method signatures | No `context.Context`; direct values | `ctx context.Context` first arg |
| Telemetry | None (zero-overhead dispatch) | OTel span per call via `bindings/runtime/tel` |
| ObjC exceptions | Not intercepted | Caught and re-raised as Go panics |
| Blocks | `purego.NewBlock` adapters from Go closures | Generated trampolines (`bindings/runtime/blocks`) |

### Runtime Layer

The runtime lives under `bindings/runtime/` — public packages imported both by the generated code and by your own application code (so consumers only ever import `bindings/…`). `bindings/runtime/purego` is imported by every generated framework package:

| Function | Purpose |
| --- | --- |
| `purego.Track(wrapper)` | Registers a Go finalizer that releases the object when the wrapper is GC'd |
| `purego.Retain(id)` / `Release(id)` | Explicit reference-count control |
| `purego.GoString(id)` | Converts an `NSString` id to a Go `string` |
| `purego.NSString(s)` | Converts a Go `string` to an autoreleased `NSString` id |
| `purego.NSErrorToError(id)` | Converts an `NSError` to a structured Go error (domain, code, reason, recovery, underlying) |
| `purego.GoCString(ptr)` | Reads a null-terminated C string address (used by block adapters) |

`bindings/runtime/purego` also re-exports the ObjC dynamic-dispatch surface (`ID`, `SEL`, `Send`, `RegisterName`, `RegisterClass`, `NewBlock`, …) so consumers performing raw message-sends never need to import the underlying `ebitengine/purego` directly.

Each generated package also carries a small `<pkg>_runtime.go`: a `sync.Once`-guarded `dlopen` of the framework dylib, per-symbol function registration with failure tracking, and the public `SymbolAvailable(symbol string) bool` probe.

The CGo library packages use `bindings/runtime/cgo` (retain/release, `RunOnMainThread`, string conversion, exception extraction) and `bindings/runtime/tel` (OTel `Call`/`RaiseIfException`/`NSErrorToError`) instead — any app that wires up an OTel exporter automatically gets a distributed trace of every C-library call.

### Generated Package Structure

Each purego ObjC framework package follows this layout:

```text
bindings/frameworks/foundation/
├── doc.go                   # Package documentation + Apple docs link
├── foundation_runtime.go    # dlopen + symbol registration + SymbolAvailable
├── foundation_enums.go      # Enum constants (+ String() methods)
├── foundation_structs.go    # C struct bindings and typedef aliases
├── foundation_externs.go    # Extern constant accessors (dlsym-backed)
├── foundation_protocols.go  # Protocol Go interfaces
├── foundation_functions.go  # Free C function wrappers (exported Go names)
└── <ClassName>.go           # One file per ObjC class (type, selectors, FromID, methods)
```

C library packages add CGo plumbing:

```text
bindings/libraries/xpc/
├── doc.go
├── cgo.go                   # LDFLAGS: -lSystem (or -framework / -l<lib>)
├── xpc_enums.go / xpc_structs.go / xpc_externs.go / xpc_protocols.go
├── xpc_functions.go         # ctx-first wrappers with OTel tracing
├── xpc_bridge_impl.m        # Block trampoline implementations
└── bridge/
    ├── xpc_bridge.h
    └── xpc_bridge.m         # Compiled with -fno-objc-arc
```

A typical generated class file (`NSString.go`):

```go
// Code generated by go-bindings-purecg. DO NOT EDIT.
//go:build darwin

package foundation

// Apple documentation: https://developer.apple.com/documentation/foundation/nsstring
type NSString struct {
    NSObject
}

// NSStringFromID wraps a raw objc.ID and registers a releasing finalizer.
func NSStringFromID(id objc.ID) *NSString { … }

func (o *NSString) Length() uint { … }

// +[NSString stringWithUTF8String:]
func NSStringStringWithUTF8String(nullTerminatedCString string) *NSString { … }
```

### Generated Code Annotations

Declarations in the generated output carry structured comments drawn from the Xcode SDK:

| Annotation | Drawn from | Example |
| --- | --- | --- |
| `// Apple documentation: <url>` | Derived per class/package | `// Apple documentation: https://developer.apple.com/documentation/foundation/nsstring` |
| `// <doc text>` | `///` or `/*!` comments in SDK headers | `// @function vmnet_start_interface @abstract …` |
| `// Deprecated: …` | `API_DEPRECATED(...)` in header | `// Deprecated: replaced by vmnet_interface_add_ip_port_forwarding_rule` |
| `// C function: <symbol>` | Export-rename provenance | `// C function: vmnet_start_interface` on `VmnetStartInterface` |

### Metadata QA

Three mechanisms keep regeneration honest:

- **`generate validate`** — structural integrity gate over committed metadata (dangling superclasses, ownership ties, enum base-type conflicts, availability anomalies). Errors fail CI.
- **Diagnostics ratchet** — `bindings --diagnostics-baseline` fails when regeneration produces a type degradation (an `unsafe.Pointer`/`objc.ID`/`objc.Block` fallback) not in the committed baseline. Fixing degradations shrinks the baseline; adding one requires a reviewed baseline edit.
- **Overrides** — `metadata/<kind>/<name>/overrides.json` applies declarative per-framework corrections (exclude, remap type, availability fix) at load time, so committed `.gometa.json` stays pure scanned output. See [docs/metadata_overrides.md](docs/metadata_overrides.md).

---

## Framework Coverage

Bindings are committed for 250 ObjC frameworks and 11 Apple C libraries discovered in the macOS 26.5 SDK. For frameworks with a Swift-only API surface (e.g. `SwiftUI`, `SwiftUICore`), the generator emits documentation-only stub packages. Coverage spans:

| Category | Examples |
| --- | --- |
| **Foundation & Core** | Foundation, CoreFoundation, CoreServices, CoreData |
| **UI & Graphics** | AppKit, QuartzCore, CoreGraphics, CoreImage, CoreText, ColorSync |
| **Audio & Video** | AVFoundation, AVFAudio, AVKit, CoreAudio, CoreMedia, VideoToolbox, AudioToolbox |
| **3D & GPU** | Metal, MetalKit, MetalPerformanceShaders, MetalFX, SceneKit, RealityKit, ModelIO |
| **Machine Learning** | CoreML, NaturalLanguage, Vision, SoundAnalysis |
| **Networking** | Network, CFNetwork, NetworkExtension, CoreBluetooth, vmnet |
| **Location & Sensors** | CoreLocation, CoreMotion, CoreHaptics, NearbyInteraction |
| **Security & Identity** | Security, LocalAuthentication, CryptoTokenKit, AuthenticationServices |
| **System & Extensions** | SystemConfiguration, SystemExtensions, ServiceManagement, ExtensionKit |
| **Media & Capture** | Photos, PhotosUI, ImageCaptureCore, ScreenCaptureKit, ReplayKit |
| **Virtualization & System** | Virtualization, Hypervisor, vmnet, EndpointSecurity (C), xpc (C), dispatch (C) |
| **C libraries** | EndpointSecurity, xpc, dispatch, oslog, sandbox, libproc, bsm, Compression, AppleArchive, xar |
| **Other** | WebKit, PDFKit, MapKit, EventKit, Contacts, StoreKit, CloudKit, and more |

Use `go run ./cmd/generate/ list` to see the exact set available in the SDK installed on your machine.

---

## Known Limitations

**Variadic functions (`va_list`)** — Objective-C methods or C functions with variadic arguments cannot be bridged safely and are excluded from generation (format-string variants are bridged with a pre-formatted string). Fixed-arity alternatives are included where the SDK provides them.

**Import cycles** — Where two frameworks have mutual class references, the cycle is detected via DFS and the lowest-weight edge is broken by degrading the typed cross-framework reference to `objc.ID`. Typed methods may be absent in those specific cross-package directions.

**Block signature limits** — Block parameters whose component types cannot cross purego's callback ABI (struct-by-value arguments, protocol interfaces, float returns) are emitted as raw `objc.Block` parameters instead of Go closures. Construct those blocks manually with `objc.NewBlock`. All such degradations are listed in `metadata/diagnostics-baseline.json`.

**Header-only symbols** — Some declared functions have no dylib export (header-inline helpers, symbols dropped in newer macOS releases). Their wrappers exist but panic when called; use the per-package `SymbolAvailable("symbol_name")` probe to check first.

**No ObjC subclassing / delegate implementation** — The purego framework packages do not currently generate Go-backed ObjC subclasses or delegate protocol implementations. Protocols are emitted as Go interfaces for typing only; APIs that require you to *implement* a delegate need a hand-written shim.

**ObjC exceptions (frameworks)** — purego framework calls do not intercept `NSException`; an uncaught ObjC exception terminates the process. The CGo library packages catch exceptions and re-raise them as Go panics.

**Go name collisions** — When two C symbols map to the same exported Go name (e.g. `__CGSizeEqualToSize` vs `CGSizeEqualToSize`, or `rb_hash` vs `rb_Hash`), the identity-named symbol wins and the transformed one is skipped with a diagnostics-baseline entry. Selector collisions on a class are disambiguated with numeric suffixes.

**Case-insensitive filenames** — On macOS APFS volumes (default, case-insensitive), two ObjC class names that differ only in case produce the same filename. The generator detects this and appends a numeric suffix (`_2`, `_3`) to disambiguate.

**Main thread requirement** — All AppKit (and generally all UI framework) calls must be dispatched on the macOS main thread. The bindings do not enforce this automatically — see [Main Thread Dispatch](#main-thread-dispatch).

**macOS only** — There is no iOS, tvOS, or watchOS support. All code is gated on `//go:build darwin`.

**arm64 preferred** — When metadata exists for multiple architectures, `arm64` takes precedence. `x86_64` (Intel) is supported but requires explicit `--arch x86_64` during a `scan`.

**Apple clang ≥ 21 (Xcode 26.x) AST compatibility** — Newer versions of Apple's clang omit `platform`/`version` data from `AvailabilityAttr` JSON nodes. The scanner falls back to parsing `API_AVAILABLE`/`API_DEPRECATED` macros from the raw SDK header source using the AST-reported source locations as anchors.

**Sparse doc comments** — Most Apple SDK headers do not use structured doc comment syntax (`///` or `/*!`). Only declarations immediately preceded by such comments receive a doc annotation.

**Importing only `bindings/…`** — Everything an external module needs is public under `bindings/`: the generated `bindings/frameworks/` and `bindings/libraries/` packages plus the shared runtime under `bindings/runtime/` (`purego`, `cgo`, `tel`, `blocks`, `callbacks`). The generator itself (`internal/codegen/…`), the scanner, and the scanned-metadata model (`internal/macosplatformmetadata`) remain in `internal/` and are not importable from outside this module — only in-repo tooling uses them.

---

## Contributing

Contributions are welcome. Please read [`CONTRIBUTING.md`](CONTRIBUTING.md) before opening a pull request.

When modifying the generator (`internal/` or `cmd/generate/`), regenerate the affected bindings (`go run ./cmd/generate/ bindings` and, if applicable, `idiomatic`) and include the updated `bindings/` output in the same PR. The diagnostics ratchet (`--diagnostics-baseline metadata/diagnostics-baseline.json`) must pass; naming rules are defined in [docs/naming.md](docs/naming.md).

---

## License

See [`LICENSE`](LICENSE).
