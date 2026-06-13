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
  - `KeepAlive(v)` — wraps `runtime.KeepAlive`; emitted as `defer runtime.KeepAlive(o)` in every generated instance method to prevent the GC from finalizing the receiver before the CGo call completes
  - `FreePtr(ptr)` — frees a `malloc`-allocated C buffer; used by generated code after copying a value-type struct return into Go memory
  - `RunOnMainThread` — executes a closure on the main GCD queue via `dispatch_sync_f`; required for all AppKit/UIKit calls
  - String conversion between `NSString *` and Go `string`
  - `ClassNameOf(ptr)` — returns the ObjC runtime class name via `object_getClass` + `class_getName`; use this to verify the concrete type before a downcast
  - `ExceptionReason` — extracts the reason string from a caught ObjC exception pointer

`bindings/runtime/tel/` wraps `go.opentelemetry.io/otel` and is imported by every generated CGo library method:
- `Call(ctx, recv, spanName)` — opens an OTel span for each ObjC method invocation and keeps the receiver alive across the CGo boundary
- `RaiseIfException(ctx, exc)` — records the exception on the active span then panics
- `NSErrorToError(ctx, ptr)` — converts an ObjC `NSError *` to a Go `error` and records it on the span

This means any app that wires up an OTel exporter automatically gets a full distributed trace of every C-library call, correlated with the application's own spans. If no provider is configured, `otel.Tracer()` returns a no-op tracer and the overhead is negligible. (The purego framework packages do not use `tel` — their dispatch is zero-overhead and uninstrumented.)

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

**`opinionated/idiomatic/`** — fully generated fluent layer (one package per framework), emitted by the frameworks pipeline (`go run ./cmd/generate/ idiomatic`). This replaced the former `opinionated/ergonomic/` layer, which has been removed.

```
opinionated/idiomatic/<name>/   # Go-friendly wrappers for each framework
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
- **Main thread**: AppKit and any UI-framework calls must be dispatched via `objc.RunOnMainThread`. The generated bindings do not do this automatically — the caller is responsible.
- **Single permitted external dependency**: `go.opentelemetry.io/otel` (and its stable transitive deps) is an intentional, foundational dependency. Every generated CGo C-library method imports `bindings/runtime/tel`, which uses OTel to trace calls and record exceptions on the active span. Do not add further external dependencies without a compelling reason reviewed by a maintainer.
- **darwin-only**: scanner and generator are gated on `//go:build darwin`. Unit tests in `internal/` that don't call Clang run on any platform.
- **Modifying the generator**: when changing `internal/` or `cmd/generate/`, re-run `go run ./cmd/generate/ bindings` and include updated `frameworks/` in the same PR. If a scanner-side change requires re-scanning a framework, run `go run ./cmd/generate/ scan --framework <Name>` so the new `.gometa.json` lands in the committed `metadata/` tree.
