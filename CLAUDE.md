# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
# Build the generator
go build ./cmd/generate/

# Run all unit tests (darwin only)
go test ./internal/...

# Run a single test
go test ./internal/codegen/libraries/emit/raw/ -run TestWriteClass

# Run acceptance tests (requires metadata/ directory)
go test ./acceptance/

# Run acceptance tests skipping the slow build step
go test ./acceptance/ -short

# Run with verbose output
go test -v ./internal/codegen/libraries/pipeline/ -run TestLoadAll

# Lint (golangci-lint v2 required)
golangci-lint run

# ── Scan ─────────────────────────────────────────────────────────────────────
# Scan a single framework (writes .gometa.json to committed metadata dir)
go run ./cmd/generate/ scan --framework Foundation

# Scan multiple frameworks
go run ./cmd/generate/ scan --framework AppKit,Foundation,CoreData

# Scan every framework in the SDK
go run ./cmd/generate/ scan --framework all

# ── Bindings ──────────────────────────────────────────────────────────────────
# Re-emit all bindings from committed metadata (no Clang needed)
# ObjC frameworks (purego) → ./bindings/frameworks/
# C libraries   (CGo)     → ./bindings/libraries/
go run ./cmd/generate/ bindings

# Re-emit with verbose diagnostic output (unsafe.Pointer degradations, cycle breaks)
go run ./cmd/generate/ bindings -v

# Re-emit and fail if any NEW type degradation appears beyond the committed baseline (CI ratchet)
go run ./cmd/generate/ bindings --diagnostics-baseline metadata/diagnostics-baseline.json

# Rewrite the diagnostics baseline (after deliberately fixing or accepting degradations)
go run ./cmd/generate/ bindings --diagnostics metadata/diagnostics-baseline.json

# Override output directories
go run ./cmd/generate/ bindings --frameworks-out ./bindings/frameworks --libraries-out ./bindings/libraries

# ── Metadata QA ───────────────────────────────────────────────────────────────
# Structural integrity checks over committed metadata (errors fail; warnings report)
go run ./cmd/generate/ validate

# Semantic API diff between two metadata trees (markdown; --json for machine output)
# e.g. before merging an SDK bump: git worktree add /tmp/old <ref>
go run ./cmd/generate/ diff --old /tmp/old/metadata --new ./metadata

# ── All-in-one ────────────────────────────────────────────────────────────────
# Scan + bindings in sequence
go run ./cmd/generate/ all --framework Foundation

# All phases for every framework in the SDK
go run ./cmd/generate/ all --framework all

# bindings from committed metadata (skip scan)
go run ./cmd/generate/ all

# ── Class Hierarchy ───────────────────────────────────────────────────────────
# Derive canonical ObjC class hierarchy from committed metadata.
# Run after 'scan' when the SDK changes; output is committed to metadata/objcclasshierarchy/.
go run ./cmd/generate/ class-hierarchy

# ── Apple developer docs ──────────────────────────────────────────────────────
# Harvest Apple's developer documentation (DocC render API) into per-framework
# appledocs.json sidecars next to the committed metadata. The pipeline loaders
# merge these into Doc fields at load time (Apple-preferred, header fallback), so
# the next 'bindings'/'idiomatic' run emits Apple's prose. See docs/appledocs.md.
go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation
go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation,AppKit --deep
go run ./scripts/tools/appledeveloperdocs fetch --framework all

# ── Main-thread (@MainActor) isolation ────────────────────────────────────────
# Harvest Swift @MainActor isolation (which APIs must run on the main thread)
# from the Swift symbol graph into per-framework mainactor.json sidecars next to
# the committed metadata. The frameworks loader merges these and propagates the
# isolation down the class hierarchy, so the next 'idiomatic' run wraps the
# affected calls in purego.Main. Re-run after an SDK bump. Requires Xcode.
go run ./scripts/tools/mainactorisolation fetch --framework AppKit
go run ./scripts/tools/mainactorisolation fetch --framework AppKit,WebKit,MapKit
go run ./scripts/tools/mainactorisolation fetch --framework all

# ── List ──────────────────────────────────────────────────────────────────────
# List all frameworks the installed SDK exposes
go run ./cmd/generate/ list

# Inspect a .gometa.json file (summary of all types)
go run ./cmd/inspect/ <path-to.gometa.json>

# Inspect a specific class within a .gometa.json file
go run ./cmd/inspect/ <path-to.gometa.json> --class NSApplication

# Regenerate the acceptance test file (default: 100 random samples, seed 0)
go run ./cmd/genacceptance/ --n 200 --seed 42
# or via Make:
make acc-generate ACC_N=200 ACC_SEED=42
```

All source files under `internal/scanner/` and `cmd/generate/` carry `//go:build darwin` — they can only be compiled on macOS.

## Architecture

This project is a **code generator** that reads macOS SDK headers via Clang and emits idiomatic Go packages. It produces the `bindings/frameworks/`, `bindings/libraries/`, and `opinionated/` trees in the same repository.

There are **two generator pipelines**, sharing the scanner and the scanned-metadata model but otherwise independent:
- **`internal/codegen/frameworks/`** — emits the purego ObjC framework packages (`bindings/frameworks/`) and the idiomatic layer (`opinionated/idiomatic/`). Pure Go, no CGo.
- **`internal/codegen/libraries/`** — emits the CGo Apple C-library packages (`bindings/libraries/`), with `.h`/`.m` bridge files. The three-phase description below traces this CGo pipeline; the frameworks pipeline mirrors it with a purego emitter.

### Three-phase pipeline

```
Clang AST dump → .gometa.json (metadata cache) → Go source files
```

**Phase 1 — Scan** (`internal/scanner/`): Invokes `xcrun clang -x objective-c -Xclang -ast-dump=json` on each framework's umbrella header. `ast.go` defines the Clang JSON AST structs. `extract.go` walks the AST and populates `macosplatformmetadata.FrameworkMeta`. `filter.go` restricts extraction to declarations from the named framework's own headers (not re-exported ones). Results are written as `<framework>-<arch>-<sdk>.gometa.json` via `internal/macosplatformmetadata/io.go`. The scanner also supports Apple C libraries that live under `{SDK}/usr/include/` rather than `System/Library/Frameworks/` (e.g. EndpointSecurity); these are registered in `knownCLibraries` in `clang.go` and produce a `LinkLib`-tagged metadata file.

**Phase 2 — Load** (`internal/codegen/libraries/pipeline/loader.go`): `LoadAll` reads every `.gometa.json` in the metadata directory into a `Registry`. The registry resolves cross-framework ownership ("which package does `NSString` live in?") using the "fewest non-zero methods wins" heuristic: the framework with the most minimal definition of a class is its canonical owner. The committed `metadata/` directory means `go run ./cmd/generate/ bindings` works without Xcode. (The frameworks pipeline has a parallel loader under `internal/codegen/frameworks/pipeline/`.)

**Phase 3 — Emit** (`internal/codegen/libraries/pipeline/generator.go` + `internal/codegen/libraries/emit/`): `GenerateBindings` topologically sorts frameworks by superclass dependency, then calls per-construct emitters for each framework. Before writing, it removes stale `.go` and bridge files from the output directory. Import cycles are detected via DFS; cycle-breaking edges substitute `unsafe.Pointer` for the typed cross-framework reference.

### Key data model (`internal/macosplatformmetadata/model.go`)

`FrameworkMeta` is the serialised/deserialised unit. It holds maps of `Class`, `Protocol`, `Enum`, `Struct`, and slices of `Function`, `Extern`, `BlockType`. `Availability` carries `API_AVAILABLE`/`API_DEPRECATED` attributes. `ForeignExtensions` captures ObjC categories that extend a class owned by a different framework (emitted as package-level functions, not methods, because Go prohibits adding methods to foreign types). `LinkLib` (optional) overrides the default `-framework <Name>` linker flag with `-l<LinkLib>` — set for C libraries that ship as plain dylibs rather than `.framework` bundles (e.g. EndpointSecurity).

### Emitters (`internal/codegen/libraries/emit/`)

One file per construct type:
- `classes.go` — one `.go` file per ObjC class; superclass chain resolved into struct embedding
- `bridge.go` — C bridge `.h`/`.m` files compiled with `-fno-objc-arc`; returned ObjC objects are `+1` retained so that `cgo.Track` can register a Go finalizer that releases them
- `interfaces.go` — one `<ClassName>able` Go interface per ObjC class, enabling duck-typed acceptance and mock implementations
- `block_trampolines.go` — generates the runtime block trampoline files written to `bindings/runtime/blocks/`: `blocks_generated.go` (Go-side `//export goCallBlock_*` functions and `MakeBlock_*` factories), `block_trampolines_generated.h`, and `block_trampolines_generated.m`
- `variadic_wrappers.go` — generates Foundation collection convenience constructors (`NSArrayOf[T]`, `NSMutableArrayOf[T]`, `NSSetOf[T]`, `NSMutableSetOf[T]`) that wrap the nil-terminated ObjC variadic pattern
- `enums.go`, `structs.go`, `externs.go`, `functions.go`, `protocols.go`, `blocks.go` — flat per-framework files
- `foreign_extensions.go` — package-level functions for categories that extend foreign classes
- `helpers.go` — shared helpers used across emitters

### Templated emission — view → render → template (ALL emitters)

Every emitter follows the same three-stage shape: a **gather** phase resolves
metadata + type info into pure-data structs (no I/O, no type decisions left for
render); a **render** phase turns those into Go source through `text/template`
files only; and a shared **assemble** phase wraps the body in the file scaffold.
No emitter assembles Go syntax with `fmt.Fprintf` anymore — the only remaining
direct `Fprintf` writes are (a) building resolved expression/comment *fragments*
that become template data, and (b) a handful of diagnostic "not bridged" comment
stubs. The view/template packages:
- raw purego frameworks: `internal/codegen/frameworks/emit/{view,render}` (+ `render/templates/*.tmpl`)
- idiomatic frameworks: `internal/codegen/frameworks/emit/idiomatic/{view,render}` (the original compiler)
- raw CGo libraries: model structs in `internal/codegen/libraries/emit/raw/model.go`, templates in `…/raw/templates/*.tmpl`, rendered via `executeTemplate`
- shared file scaffold: `internal/codegen/shared/fileasm` — `Assemble(File)` + `file.tmpl` + the two import groupers (`ImportLinesStdlibMod`, `ImportLinesStdlibExternalInternal`)

**Two gofmt invariants every template author must respect** (a violation silently
changes generated bytes — caught only by the regen+diff gate):
1. **Comment indentation.** gofmt's doc-comment reformatter smart-quotes a
   *column-0* comment (`` ``x`` `` → `“x“`) but leaves an *already-indented* one
   verbatim. So a comment that the original emitter wrote tab-indented (e.g. an
   enum member's doc inside a `const (…)` block) must be emitted **with its
   leading tab** in the gather, not at column 0.
2. **One-line vs multiline bodies.** gofmt preserves a one-line `if x { y }` and a
   multiline one as written. Match the original exactly — e.g. the object-return
   retain guard `if _ret != 0 { _ret.Send(…) }` is templated single-line, while
   the NSError guards are multiline.

The gate for any emitter change: regenerate (`go run ./cmd/generate/ bindings`
and `… idiomatic`) and confirm `git diff` of `bindings/` + `opinionated/` is
empty unless the change is intended.

### Type mapping (`internal/codegen/libraries/typemap/mapper.go`)

`Mapper.GoType` converts ObjC `qualType` strings (e.g. `NSArray<NSString *> *`) to Go type strings (e.g. `*foundation.NSArray[*foundation.NSString]`). It is context-sensitive: it knows the current class (for `instancetype`), which classes use Go generics, and which cross-framework imports are blocked (cycle prevention). Resolved cross-framework types are collected into the caller's `UsedImports` map as a side effect.

### Naming (`internal/codegen/libraries/naming/naming.go`)

- `MethodName` converts ObjC selectors to exported Go method names (`objectAtIndex:` → `ObjectAtIndex`; `UsingBlock` suffix → `Using`)
- `BridgeFuncName` produces the C bridge symbol (`foundation_NSString_stringWithCString`)
- `ArgName` lowercases and escapes Go reserved words
- `PackageName` lowercases the framework name

### Runtime (`bindings/runtime/`)

The runtime layer lives under `bindings/runtime/` — **public** packages imported both by the generated code and by external consumers (so consuming projects only ever import `bindings/…`). There are two runtimes, matching the two pipelines:

- **`bindings/runtime/purego`** (package `purego`) — the runtime for every generated framework package (`bindings/frameworks/`). Pure-Go (over `github.com/ebitengine/purego`): `Track`/`Retain`/`Release`, `GoString`/`NSString`, `NSErrorToError` (structured `objcerrors.ObjCError`), `GoCString`. It also re-exports the ObjC dynamic-dispatch surface (`ID`, `SEL`, `Send`, `RegisterName`, `RegisterClass`, `NewBlock`, …) so consumers never import `ebitengine/purego` directly. The `objcerrors` subpackage holds the structured error type.
- **`bindings/runtime/cgo`** (package `cgo`) — the darwin-only CGo runtime for the C-library packages (`bindings/libraries/`, compiled with `-fno-objc-arc`):
  - `Retain`/`Release`/`Track` — explicit ObjC retain/release; `Track` registers a Go finalizer so the GC automatically releases `+1`-retained objects
  - `KeepAlive(v)` — wraps `runtime.KeepAlive`; emitted as `defer cgo.KeepAlive(o)` in every generated instance method (and for each ObjC-object argument) to prevent the GC from finalizing the receiver before the CGo call completes
  - `FreePtr(ptr)` — frees a `malloc`-allocated C buffer; used by generated code after copying a value-type struct return into Go memory
  - `RunOnMainThread` — executes a closure on the main GCD queue via `dispatch_sync_f`; required for all AppKit/UIKit calls
  - String conversion between `NSString *` and Go `string`
  - `ClassNameOf(ptr)` — returns the ObjC runtime class name via `object_getClass` + `class_getName`; use this to verify the concrete type before a downcast
  - `ExceptionInfoFromPtr`/`ExceptionReason` — extract structured fields (name, reason) from a caught ObjC exception pointer
  - `RaiseIfException(exc)` — converts a caught ObjC `NSException *` into a Go panic (no-op when nil); calls the `OnException` hook first
  - `NSErrorToError(ptr)` — converts an ObjC `NSError *` to a Go `error` (and releases the pointer)
  - `OnException` / `OnCallbackPanic` — optional package-level hooks an application can set at startup to route ObjC exceptions and recovered callback panics to its structured logger

Generated CGo library functions and methods take **no** `context.Context` parameter and have **no** telemetry: they call the C bridge directly, then `cgo.RaiseIfException(_exc)` (and `cgo.NSErrorToError(_nsErr)` for `NSError`-returning calls), keeping the receiver and object arguments alive via `defer cgo.KeepAlive(...)`. This matches the zero-overhead, uninstrumented dispatch of the purego framework packages.

### Block trampoline runtime (`bindings/runtime/blocks/`)

ObjC blocks passed to Go are managed through a closure registry in this package. `BlockRegister(fn)` stores a Go closure in a `sync.Map` and returns a `uint64` handle. The generated C trampoline receives this handle and calls back into Go via an `//export goCallBlock_*` function, which looks up and invokes the closure. `FreeBlock` and the `//export goBlockUnregister`/`goBlockRetain` functions manage the block's lifetime when ObjC copies or disposes it. The `.go`, `.h`, and `.m` files in this package that end in `_generated` are written by the `block_trampolines.go` emitter — do not edit them by hand.

### Swift-only frameworks (`internal/swift/`)

Some Apple frameworks (e.g. RealityKit, SwiftUI extensions) have no ObjC surface area — their API is entirely Swift. The generator detects these via `SwiftOnly: true` in `FrameworkMeta` and takes a different path:

1. `internal/swift/parser/` — parses `.swiftinterface` module definition files, extracting enums, value structs, and error types into `internal/swift/meta.FrameworkMeta`.
2. `internal/swift/emit/` — writes Go type files from the parsed Swift metadata (enums, error codes, value types only; methods cannot be bridged without a C shim).

Swift-only frameworks that cannot be parsed (missing `.swiftinterface`) receive a documentation-only stub package. These are not bridgeable via CGo and should not be expected to have callable methods.

### Generated package layout

ObjC frameworks emit to `frameworks/<name>/`, Apple C libraries emit to `libraries/<name>/`:

```
frameworks/<name>/          (ObjC frameworks)
libraries/<name>/           (Apple C libraries, e.g. EndpointSecurity)
├── doc.go
├── cgo.go                  # -framework <Name> for ObjC; -l<lib> for C libraries
├── <name>_enums.go
├── <name>_structs.go
├── <name>_externs.go
├── <name>_protocols.go
├── <name>_interfaces.go    # <ClassName>able Go interfaces for duck-typing
├── <name>_functions.go
├── <name>_foreign_extensions.go      (if any)
├── <name>_bridge_impl.m              # block trampoline implementations
├── <ClassName>.go                    # one file per ObjC class
└── bridge/
    ├── <name>_bridge.h
    └── <name>_bridge.m               # compiled with -fno-objc-arc
```

All generated files begin with `// Code generated by go-bindings-codegen. DO NOT EDIT.` and `//go:build darwin`.

### Opinionated layers

Two generated opinionated layers sit on top of the raw `frameworks/` and `libraries/` packages:

**`opinionated/library/`** — hand-crafted and generated quality-of-life helpers:

```
opinionated/library/
├── foundation/          # hand-crafted: FileURL, NSDataToBytes, StringFromGo, FileHandle, Progress, OSVersion
├── corefoundation/      # hand-crafted: CGSize/CGPoint/CGRect constructors and predicates
├── bsd/                 # hand-crafted: EtherAddr ↔ string helpers
├── appkit/              # hand-crafted: screen helpers + generated *_generated.go
├── virtualization/      # hand-crafted: capabilities, state, network, storage, bootloader,
│                        #               machine_id, macos, display, installer, rosetta,
│                        #               restore_image, socket, constants, create
│                        #               + generated *_generated.go
└── <other frameworks>/  # generated *_generated.go only (async wrappers, typed slices, spec types)
```

**`opinionated/idiomatic/`** — fully generated fluent layer, emitted by the frameworks pipeline
(`go run ./cmd/generate/ idiomatic`). This replaced the former `opinionated/ergonomic/` layer,
which has been removed. The emitter is a compiler-style pipeline: a resolution pass builds a
pure-data IR (the `view` package under `internal/codegen/frameworks/emit/idiomatic/view/`) and a
render pass turns it into Go source through `text/template` files only (`…/render/templates/`) —
no Go syntax is string-built, and imports are computed from resolved types (not scanned from the
output). The emitter source is split by construct across the `idiomatic` package
(`classes.go`, `constructors.go`, `setters.go`, `methods.go`, `typeresolve.go`, `docs.go`,
`naming.go`, …). Wrappers embed their same-framework base (promoting its methods), abstract-base
setters accept a **sealed** provider interface (`<Base>Provider` with an unexported marker), and
the layer is hermetic (never imports `bindings/frameworks`).

```
opinionated/idiomatic/framework/<name>/    # Go-friendly wrappers for each ObjC framework
opinionated/idiomatic/libraries/<name>/    # wrappers for each Apple C library
```

`opinionated/custom/` holds additional hand-crafted packages.

Rules for the opinionated layers:
- All hand-crafted files use `//go:build darwin` and import raw frameworks as `raw "…/bindings/frameworks/<name>"`.
- Cross-package imports inside `opinionated/library/` always use the full module path `github.com/deploymenttheory/go-bindings-macosplatform/opinionated/library/<name>`.
- The generator **never deletes** hand-crafted files in `opinionated/library/`; only `*_generated.go` files are regenerated.
- All files under `opinionated/idiomatic/` are generated — do not hand-edit them.
- **Never modify `bindings/frameworks/` or `bindings/libraries/` as part of opinionated work.** Raw generated code must remain untouched.

### Metadata cache (`metadata/`)

Pre-built `.gometa.json` files are committed here, split by type:

```
metadata/
├── frameworks/<name>/<Framework>-arm64-<sdk>.gometa.json   (ObjC frameworks)
│   └── carbon/hitoolbox/HIToolbox-arm64-26.5.gometa.json   (sub-frameworks one level deeper)
├── frameworks/<name>/overrides.json                        (optional declarative fixups — see docs/metadata_overrides.md)
├── libraries/<name>/<Library>-arm64-<sdk>.gometa.json      (Apple C libraries)
├── clibraries.json                                         (Apple C library registry: name → link_lib/header/header_dir)
├── scanconfig.json                                         (per-framework scan config, e.g. Foundation's usr/include/objc/)
└── diagnostics-baseline.json                               (known type degradations; CI fails on new entries)
```

These allow `go run ./cmd/generate/ bindings` regeneration without Xcode. The loader globs both `metadata/frameworks/` and `metadata/libraries/` automatically.

Every `.gometa.json` carries `schema_version` (checked by both `meta.Read` implementations — bump `meta.CurrentSchemaVersion` when `FrameworkMeta` changes incompatibly, then re-scan) plus `clang_version`/`xcode_version` toolchain provenance recorded at scan time.

### Metadata QA and corrections

- **`generate validate`** — structural integrity gate over the committed metadata (dangling superclasses, ownership ties, enum base-type conflicts, availability anomalies, mixed-SDK trees). Runs in CI; errors fail.
- **`generate diff`** — semantic API diff between two metadata trees; use it to make SDK-bump PRs reviewable instead of eyeballing 100+ MB of JSON.
- **Diagnostics ratchet** — `bindings --diagnostics-baseline metadata/diagnostics-baseline.json` fails when regeneration produces a type degradation not in the committed baseline. Fixing degradations shrinks the baseline; adding one requires a reviewed baseline edit (`--diagnostics` rewrites it).
- **Overrides** (`metadata/<kind>/<name>/overrides.json`) — declarative per-framework corrections (exclude, remap type, force bitmask, availability fix, link lib) applied at load time so committed metadata stays pure scanned output. Format: [`docs/metadata_overrides.md`](docs/metadata_overrides.md). Prefer an override for a local defect; change scanner/typemap code only for systemic ones.

## Naming Standards

The full naming contract lives in [`docs/naming.md`](docs/naming.md). The rules below are
non-negotiable — violations block PRs.

| Concept | Rule |
|---------|------|
| **Boolean fields** | `Is*` prefix always (`IsNullable`, `IsReadOnly`). No bare adjectives, no `Has*`/`Needs*`/`In*`. |
| **ObjC parameters** | `Param`/`Params`/`param` everywhere. Never `Arg`/`Args`/`arg`. Field `Direction` not `Modifier`. |
| **Return descriptor** | Struct is `meta.ReturnType`. Field on Method/Function is `Return`. Local var is `retType`. |
| **Framework variable** | Always `framework *meta.FrameworkMeta`. Never `fm`, `fw`, `f`. |
| **Go package variable** | `packageName`, `packagePath`, `packageAlias`. Never `pkg`. |
| **Selector string** | Always `selector`. Never `sel` or `methodSelector`. |
| **Scanner verbs** | `scan*` = AST walk · `parse*` = string/YAML · `make*` = struct assembly |
| **Emitter tiers** | `Emit*` exported entry-points · `write*` I/O helpers · `build*` model constructors |
| **Template structs** | `*Model` suffix for all structs in `emit/raw/model.go` |
| **Registry maps** | `*Index` suffix (`ClassIndex`, `OwnerIndex`, `EnumIndex`, …) |
| **Loop variables** | Full word: `class`, `method`, `param`, `protocol`. Never `cls`, `m`, `p`. |
| **Buffers** | `w` io.Writer param · `buf` bytes.Buffer · `sb` strings.Builder |
| **Bridge term** | "C bridge layer" (files) · "bridge function" (trampoline) · "type mapping" (never "type bridging") |

## Important constraints

- **ARC disabled**: all bridge `.m` files use `-fno-objc-arc`. Do not mix ARC code with the bridges.
- **Main thread**: Apple isolates AppKit and other UI APIs to Swift's `@MainActor` (the macro `NS_SWIFT_UI_ACTOR`). The **idiomatic** layer (`opinionated/idiomatic/framework/`) now wires this automatically: methods, setters, and constructors of an `@MainActor`-isolated class — and every subclass that inherits the isolation (e.g. `MKMapView`, `PDFView` via `NSView`) — are wrapped in `purego.Main`, which runs the call on the main thread (inline when already there). The isolation is harvested from the Swift symbol graph into committed `metadata/frameworks/<name>/mainactor.json` sidecars (see `go run ./scripts/tools/mainactorisolation`), merged and propagated down the class hierarchy at load time. The **raw** bindings (`bindings/frameworks/`) are unchanged — callers there are still responsible for `purego.Main`/`objc.RunOnMainThread`. Queue-based frameworks (Virtualization, Core Data) carry no `@MainActor` and are correctly left unwrapped.
- **Single permitted external dependency**: `github.com/ebitengine/purego` is the only non-stdlib dependency, used by the purego framework runtime. The CGo C-library layer has no external runtime dependency (it is pure CGo over `bindings/runtime/cgo`). Do not add further external dependencies without a compelling reason reviewed by a maintainer. (OpenTelemetry was previously a foundational dependency of the CGo libraries layer via `bindings/runtime/tel`; that package and the dependency have been removed — library calls are now `context`-free and uninstrumented.)
- **darwin-only**: scanner and generator are gated on `//go:build darwin`. Unit tests in `internal/` that don't call Clang run on any platform.
- **Modifying the generator**: when changing `internal/` or `cmd/generate/`, re-run `go run ./cmd/generate/ bindings` and include updated `frameworks/` in the same PR. If a scanner-side change requires re-scanning a framework, run `go run ./cmd/generate/ scan --framework <Name>` so the new `.gometa.json` lands in the committed `metadata/` tree.
