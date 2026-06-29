# Extraction Workflow

This page explains how `go-bindings-macosplatform` turns macOS SDK header files into idiomatic Go packages — from raw C header text all the way to type-safe Go bindings with CGo bridges.

The pipeline runs in three sequential phases:

```text
Phase 1 — Scan     Clang AST → .gometa.json (one file per framework)
Phase 2 — Load     .gometa.json files → Registry (cross-framework type index)
Phase 3 — Emit     Registry → Go source files + C bridge files
```

---

## Contents

- [Glossary](#glossary)
- [Why Clang?](#why-clang)
- [Phase 1 — Scan: Clang AST → `.gometa.json`](#phase-1--scan-clang-ast--gometajson)
  - [What is a Clang AST?](#what-is-a-clang-ast)
  - [Umbrella Headers](#umbrella-headers)
  - [The Scanner Invocation](#the-scanner-invocation)
  - [Filtering to the Right Framework](#filtering-to-the-right-framework)
  - [What Gets Extracted](#what-gets-extracted)
  - [Availability Parsing](#availability-parsing)
  - [Output: `.gometa.json`](#output-gometajson)
- [Phase 2 — Load: `.gometa.json` → Registry](#phase-2--load-gometajson--registry)
  - [The Registry](#the-registry)
  - [Ownership Heuristic](#ownership-heuristic)
  - [Architecture Preference](#architecture-preference)
- [Phase 3 — Emit: Registry → Go Source](#phase-3--emit-registry--go-source)
  - [Topological Sort](#topological-sort)
  - [Stale-File Cleanup](#stale-file-cleanup)
  - [Import Cycle Detection](#import-cycle-detection)
  - [Per-Construct Emitters](#per-construct-emitters)
  - [Type Mapping](#type-mapping)
  - [Naming Conventions](#naming-conventions)
- [The Metadata Cache](#the-metadata-cache)

---

## Glossary

**AST (Abstract Syntax Tree)** — A tree representation of source code produced by a compiler's parser. Each node in the tree is a language construct: a function declaration, a class definition, an enum, a variable. The AST carries exact source locations (file and line number) for every node.

**Clang** — The C/C++/Objective-C compiler front-end used by Apple's toolchain. Clang can dump the AST it produces as JSON (`-ast-dump=json`), which is what the scanner uses.

**ObjC / Objective-C** — Apple's object-oriented extension of C, used to write the macOS and iOS SDKs. ObjC classes are declared with `@interface` / `@end`; methods are identified by their *selector*.

**Selector** — An Objective-C method identifier. A selector includes the method name and all argument labels separated by colons: `objectAtIndex:` (one argument), `dictionaryWithObject:forKey:` (two arguments). The selector uniquely identifies a method in ObjC's runtime dispatch table.

**Framework** — A macOS bundle that packages a library's headers, compiled binary, and resources together. Each framework has a name (e.g. `Foundation`) and a single *umbrella header*.

**Umbrella header** — A single `<Framework/Framework.h>` file that `#include`s all other public headers in the framework. Clang-dumping the umbrella header produces an AST that covers the entire public API.

**CGo** — Go's mechanism for calling C functions from Go (and Go functions from C). Generated bindings use CGo to bridge Go method calls through thin C functions to the Objective-C runtime.

**`.gometa.json`** — The intermediate metadata file produced by Phase 1 and consumed by Phases 2 and 3. It contains a serialised `FrameworkMeta` value: all classes, protocols, enums, structs, functions, extern constants, block types, source locations, and availability annotations for one framework on one architecture.

**Registry** — The in-memory index built in Phase 2. It maps every known class name, protocol, enum, and struct to its owning framework, enabling the emitter to produce precise import statements rather than falling back to `unsafe.Pointer`.

---

## Why Clang?

An alternative approach would be to parse the SDK header files as text: scan for `@interface`, `@protocol`, `typedef enum`, etc. using regular expressions or a hand-written parser.

The Clang AST approach is more robust for several reasons:

1. **Macros are expanded.** The SDK headers use hundreds of C macros (`NS_AVAILABLE`, `NS_ENUM`, `NS_OPTIONS`, `API_DEPRECATED_WITH_REPLACEMENT`, `NS_DESIGNATED_INITIALIZER`, …). A text parser would need to understand every macro. Clang expands them before producing the AST, so the scanner sees the final resolved declarations.

2. **`#include` resolution is complete.** A framework's umbrella header includes other headers which include yet others. Clang follows all of them and produces a single unified AST. The scanner just needs to filter nodes by their source file to isolate the framework's own declarations.

3. **Exact source locations.** Every AST node carries the precise file and line where it was declared. The scanner uses these to read availability annotations (`API_AVAILABLE(macos(10.15))`) directly from the raw header source, and to associate doc comments with their declarations.

4. **Semantic attributes.** Clang attaches semantic information to AST nodes that isn't visible in the source text: whether a method is a designated initialiser, whether a return value must not be discarded, whether a parameter escapes the call, whether an enum is a bitmask. These drive the generated Go annotations.

---

## Phase 1 — Scan: Clang AST → `.gometa.json`

### What is a Clang AST?

When Clang parses an Objective-C file it builds a tree where each node represents a language construct. The top-level node is a `TranslationUnitDecl` (the whole file). Its children are declarations: `ObjCInterfaceDecl` for a class, `EnumDecl` for an enum, `FunctionDecl` for a free function, etc. Each node contains its children (methods on a class, members of an enum) and carries attributes (`AvailabilityAttr`, `ObjCDesignatedInitializerAttr`, …).

A tiny example — the JSON for part of `NSString`'s interface:

```json
{
  "kind": "ObjCInterfaceDecl",
  "name": "NSString",
  "loc": {
    "file": "/…/Foundation.framework/Headers/NSString.h",
    "line": 23
  },
  "super": { "name": "NSObject" },
  "protocols": [
    { "name": "NSCopying" },
    { "name": "NSMutableCopying" },
    { "name": "NSSecureCoding" }
  ],
  "inner": [
    {
      "kind": "ObjCMethodDecl",
      "name": "length",
      "instance": true,
      "returnType": { "qualType": "NSUInteger" },
      "inner": [
        { "kind": "AvailabilityAttr" }
      ]
    },
    {
      "kind": "ObjCMethodDecl",
      "name": "stringWithUTF8String:",
      "instance": false,
      "returnType": { "qualType": "instancetype" }
    }
  ]
}
```

The scanner reads this tree and populates Go structs. It does not need to understand C preprocessor macros or `#include` chains — Clang has already resolved all of that.

### Umbrella Headers

Every macOS framework provides a single umbrella header named after the framework:

```text
/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/
  Developer/SDKs/MacOSX.sdk/System/Library/Frameworks/
    Foundation.framework/Headers/Foundation.h     ← umbrella header
    AppKit.framework/Headers/AppKit.h
    Virtualization.framework/Headers/Virtualization.h
    …
```

The umbrella header `#include`s all the framework's other public headers, so dumping it gives a complete picture of the framework's public API in a single AST.

### The Scanner Invocation

For each framework the scanner runs:

```sh
xcrun clang \
    -x objective-c \
    -Xclang -ast-dump=json \
    -fsyntax-only \
    -isysroot /path/to/MacOSX.sdk \
    /path/to/Framework.framework/Headers/Framework.h
```

- `-x objective-c` — parse as Objective-C (not plain C).
- `-Xclang -ast-dump=json` — pass the `-ast-dump=json` flag directly to the Clang front-end, causing it to emit the AST as JSON on stdout instead of compiling.
- `-fsyntax-only` — stop after parsing; do not compile or link.
- `-isysroot` — point Clang at the correct SDK headers.

`xcrun` automatically resolves the path to the active Xcode toolchain's `clang` binary. The scanner captures stdout and deserialises it as a tree of `ASTNode` structs.

### Filtering to the Right Framework

Clang follows every `#include` transitively. The AST for `Foundation.h` contains declarations from `Foundation`, but also from `CoreFoundation`, `objc/NSObject.h`, `usr/include/`, and other headers that Foundation itself imports.

`filter.go` restricts extraction to nodes whose source file (`node.Loc.ResolvedFile()`) is inside the target framework's own `Headers/` directory:

```text
Keep:   /…/Foundation.framework/Headers/NSString.h
Keep:   /…/Foundation.framework/Headers/NSArray.h
Skip:   /…/CoreFoundation.framework/Headers/CFString.h
Skip:   /usr/include/objc/NSObject.h
```

This ensures that each `.gometa.json` file contains only the framework's own declarations, not those of its dependencies.

### What Gets Extracted

The scanner walks the AST and populates a `meta.FrameworkMeta` struct. Each ObjC construct maps to a Go type:

| ObjC construct | AST node kind | Metadata type |
| --- | --- | --- |
| Class (`@interface`) | `ObjCInterfaceDecl` | `meta.Class` |
| Protocol (`@protocol`) | `ObjCProtocolDecl` | `meta.Protocol` |
| Category (`@interface X (Category)`) | `ObjCCategoryDecl` | extends `meta.Class` or becomes a `ForeignExtension` |
| Enum (`NS_ENUM`, `NS_OPTIONS`, `typedef enum`) | `EnumDecl` | `meta.Enum` |
| Struct (`typedef struct`) | `RecordDecl` | `meta.Struct` |
| Free function | `FunctionDecl` | `meta.Function` |
| Extern variable/constant | `VarDecl` | `meta.Extern` |
| Block type (`typedef void (^Block)(...)`) | `TypedefDecl` wrapping a block | `meta.BlockType` |

Every extracted declaration carries:

- `SDKFile` — relative header path, e.g. `Foundation/NSString.h`
- `SDKLine` — line number in that header
- `Doc` — extracted doc comment if the header uses `///` or `/*!` syntax
- `Availability` — macOS introduced/deprecated/obsoleted versions and deprecation message

### Availability Parsing

Apple's SDK headers use macros to annotate when an API was introduced, deprecated, or removed:

```objc
// Foundation/NSFileManager.h
- (BOOL)createDirectoryAtURL:(NSURL *)url
    withIntermediateDirectories:(BOOL)createIntermediates
    attributes:(NSDictionary<NSFileAttributeKey,id> *)attributes
    error:(NSError **)error
    API_AVAILABLE(macos(10.7));

- (NSString *)currentDirectoryPath
    API_DEPRECATED("Use FileManager.currentDirectoryURL", macos(10.0, 26.0));
```

Older Clang versions embedded these version numbers directly in `AvailabilityAttr` JSON nodes. **Apple Clang 21+ (Xcode 26.x) no longer does this** — the `AvailabilityAttr` node exists but carries no platform or version data.

The scanner falls back to parsing availability annotations directly from the raw header source text, using the AST-reported line numbers as anchors:

1. Record the line number of the declaration from the AST.
2. Read the raw header file and search the lines near that position for `API_AVAILABLE(macos(...))` / `API_DEPRECATED(...)` / `API_DEPRECATED_WITH_REPLACEMENT(...)`.
3. Extract the version strings with a regex.

This two-path approach handles both old and new Clang versions transparently.

### Output: `.gometa.json`

The scanner writes one file per framework per architecture:

```text
metadata/
├── foundation/
│   └── Foundation-arm64-26.5.gometa.json
├── appkit/
│   └── AppKit-arm64-26.5.gometa.json
├── virtualization/
│   └── Virtualization-arm64-26.5.gometa.json
└── carbon/hitoolbox/
    └── HIToolbox-arm64-26.5.gometa.json   ← sub-framework
```

The file is the JSON serialisation of `meta.FrameworkMeta`. A truncated example:

```json
{
  "framework": "Foundation",
  "sdk_version": "26.5",
  "arch": "arm64",
  "classes": {
    "NSString": {
      "name": "NSString",
      "superclass": "NSObject",
      "sdk_file": "Foundation/NSString.h",
      "sdk_line": 23,
      "methods": [
        {
          "selector": "length",
          "is_instance": true,
          "return_type": "NSUInteger",
          "availability": { "macos_introduced": "10.0" }
        }
      ]
    }
  },
  "enums": { … },
  "structs": { … },
  "functions": [ … ]
}
```

---

## Phase 2 — Load: `.gometa.json` → Registry

### The Registry

`pipeline/loader.go` reads every `.gometa.json` in the metadata directory into a `Registry`. The registry is the single source of truth for cross-framework type resolution:

```go
type Registry struct {
    Frameworks         []*meta.FrameworkMeta
    KnownClasses       map[string]bool           // all class names across all frameworks
    GenericClasses     map[string]bool           // classes with ObjC generic params
    FrameworkOwner     map[string]string         // class name → owning framework
    KnownProtocols     map[string]string         // protocol name → owning framework
    KnownEnums         map[string]string         // enum name → owning framework
    KnownStructs       map[string]string         // struct name → owning framework
    KnownTypedefs      map[string]string         // typedef name → target type
    ModulePrefix       string                    // Go module import path prefix
}
```

When the emitter encounters `NSArray<NSString *> *` as a method return type, it queries `FrameworkOwner["NSString"]` to find `"Foundation"`, then generates the import `github.com/…/frameworks/foundation` in the produced Go file. Without the registry, every cross-framework type reference would have to degrade to `unsafe.Pointer`.

### Ownership Heuristic

The same class name can appear in multiple frameworks' `.gometa.json` files. `NSString` appears in both `Foundation` (its canonical definition with hundreds of methods) and in `CoreFoundation` (a minimal forward declaration with zero methods, because CoreFoundation imports Foundation headers for its toll-free bridging).

The registry uses the **"fewest non-zero methods wins"** heuristic to determine canonical ownership: the framework that sees the *most minimal declaration* of a class is taken as its owner. In the `NSString` example, `CoreFoundation` has fewer methods → it "owns" more minimal → but `Foundation`'s richer declaration wins by having fewer *non-zero* properties to indicate it is the canonical home.

More precisely: the framework whose metadata entry for a class has the fewest methods among all entries for that class is treated as its authoritative owner. This correctly identifies re-exported forward declarations (which have zero or very few methods) vs. the real definition.

### Architecture Preference

When metadata exists for multiple architectures (e.g. both `arm64` and `x86_64`), the loader prefers `arm64`. Intel metadata is used as a fallback only when arm64 is not available.

---

## Phase 3 — Emit: Registry → Go Source

`pipeline/generator.go` drives the emit phase. It iterates over all frameworks in topological order and calls the per-construct emitters for each one.

### Topological Sort

A framework's generated Go package imports the packages of frameworks it depends on (for superclass struct embedding, method argument types, etc.). Go's build system requires that dependencies compile before dependants.

The generator performs a topological sort of frameworks ordered by their superclass dependency graph before emitting. `Foundation` compiles before `AppKit`; `AppKit` compiles before `Virtualization`; and so on.

### Stale-File Cleanup

Before writing new output, the generator removes all existing `.go` and bridge (`.h`, `.m`) files from the framework's output directory. This prevents stale declarations from accumulating when a class is renamed, removed, or moved to a different framework between generations.

### Import Cycle Detection

Two frameworks can have mutual class references: framework A has a class that returns a type from framework B, and framework B has a class that returns a type from framework A. This creates a circular import, which Go's build system does not allow.

The generator detects cycles via depth-first search on the import graph. For each detected cycle, it finds the lowest-weight edge (the cross-framework reference used least often) and breaks it by substituting `unsafe.Pointer` for the typed cross-package reference in that direction. The affected method still works — callers get an `unsafe.Pointer` they can wrap with the target type's `WithPtr` constructor — but the cycle is eliminated.

The verbose flag (`-v`) prints all cycle-breaking substitutions:

```text
[cycle] AppKit → Foundation → AppKit: substituting unsafe.Pointer for NSAffineTransform
```

### Per-Construct Emitters

Each ObjC construct type has its own emitter file:

| Emitter | Input | Output |
| --- | --- | --- |
| `emit/classes.go` | `meta.Class` | `<ClassName>.go` per class; `<framework>_interfaces.go` (duck-typing interfaces) |
| `emit/bridge.go` | All methods and functions | `bridge/<framework>_bridge.h` + `bridge/<framework>_bridge.m` |
| `emit/enums.go` | `meta.Enum` | `<framework>_enums.go` |
| `emit/structs.go` | `meta.Struct` | `<framework>_structs.go` |
| `emit/externs.go` | `meta.Extern` | `<framework>_externs.go` |
| `emit/functions.go` | `meta.Function` | `<framework>_functions.go` |
| `emit/protocols.go` | `meta.Protocol` | `<framework>_protocols.go` |
| `emit/foreign_extensions.go` | ObjC categories on foreign types | `<framework>_foreign_extensions.go` |
| `emit/block_trampolines.go` | All block types | `internal/blocks/blocks_generated.go` + `.h` + `.m` |
| `emit/variadic_wrappers.go` | — | Convenience constructors for `NSArray`, `NSSet`, etc. |

**`emit/bridge.go`** produces the C bridge files. Each ObjC method call is wrapped in a thin C function:

```c
// bridge/foundation_bridge.h
void* foundation_NSString_stringWithUTF8String(const char* str, void** exc);

// bridge/foundation_bridge.m  (compiled with -fno-objc-arc)
void* foundation_NSString_stringWithUTF8String(const char* str, void** exc) {
    @try {
        return [NSString stringWithUTF8String:str];
    }
    @catch (NSException *e) {
        *exc = (__bridge_retained void*)e;
        return nil;
    }
}
```

Every bridge call wraps the ObjC dispatch in `@try`/`@catch`. A caught exception is stored in the `exc` out-parameter; the generated Go side calls `tel.RaiseIfException(ctx, exc)` to convert it to a Go panic.

**`emit/classes.go`** produces one `.go` file per class. Superclass relationships are modelled using struct embedding:

```go
// NSMutableString embeds NSString which embeds NSObject
type NSMutableString struct {
    NSString  // struct embedding — all NSString methods are promoted
}
```

The generated `New*` constructor registers a Go finalizer via `objc.Track` so the ObjC object is released when the Go wrapper is garbage-collected.

**`emit/block_trampolines.go`** generates the runtime bridge between Go closures and ObjC blocks. An ObjC block is a C struct with a function pointer. For each distinct block signature (`void (^)(id, NSUInteger, BOOL *)`) the emitter produces:

- A Go `//export goCallBlock_*` function that looks up and invokes a registered Go closure by numeric handle.
- A C trampoline `.m` function that receives the handle from ObjC and calls back into Go.
- A `MakeBlock_*` factory in Go that registers a closure and returns the ObjC block struct.

This allows generated Go methods to accept plain Go `func` values for block parameters without any manual CGo ceremony.

### Type Mapping

`typemap/mapper.go` converts ObjC `qualType` strings (the raw C type expression) to Go type strings. This is context-sensitive:

| ObjC `qualType` | Go type | Notes |
| --- | --- | --- |
| `NSString *` | `*foundation.NSString` | Cross-framework: imports `foundation` |
| `NSArray<NSString *> *` | `*foundation.NSArray[*foundation.NSString]` | Generic container with type parameter |
| `instancetype` | `*ClassName` | Resolved to the current class |
| `NSInteger` | `int64` | Primitive mapping |
| `BOOL` | `bool` | |
| `SEL` | `unsafe.Pointer` | ObjC selector, opaque |
| `id` | `objc.Object` | Untyped ObjC reference |
| `NSError **` | triggers `error` return value | Out-parameter becomes Go `error` |
| (cycle-broken edge) | `unsafe.Pointer` | Import cycle prevention |

Cross-framework types are recorded in a `UsedImports` map as a side effect of `GoType()`. The caller (the class emitter) collects all imports needed by the file this way, then writes a single `import (...)` block at the top of the generated file.

### Naming Conventions

`naming/naming.go` converts ObjC identifiers to Go identifiers:

**Selectors → method names:**

| ObjC selector | Go method name | Rule |
| --- | --- | --- |
| `objectAtIndex:` | `ObjectAtIndex` | Strip colons, capitalise each segment |
| `setTitle:` | `SetTitle` | Prefix `set` retained as-is |
| `enumerateObjectsUsingBlock:` | `EnumerateObjectsUsing` | `Block` suffix stripped |
| `initWithContentsOfURL:options:error:` | `InitWithContentsOfURLOptionsError` | All segments joined |
| `init` | `Init` | No colons → simple capitalise |

When two selectors on the same class produce the same Go name, the generator appends the argument count (`_2`, `_3`), with a further sequential suffix if needed.

**Bridge function names:**

```text
<lowercased_framework>_<ClassName>_<camelCaseSelector>

foundation_NSString_stringWithUTF8String
appkit_NSWindow_makeKeyAndOrderFront
```

**Package names:** the lowercased framework name — `foundation`, `appkit`, `corefoundation`.

**Argument names:** lowercased first segment of the selector label. Go reserved words are escaped (`type` → `type_`, `func` → `func_`).

---

## The Metadata Cache

All 258 pre-built `.gometa.json` files are committed to the repository under `metadata/`:

```text
metadata/
├── foundation/Foundation-arm64-26.5.gometa.json
├── appkit/AppKit-arm64-26.5.gometa.json
├── …
└── carbon/hitoolbox/HIToolbox-arm64-26.5.gometa.json
```

This serves two purposes:

1. **`--skip-scan` regeneration** — Phases 2 and 3 can run without Xcode or Clang. Anyone can regenerate all 251 framework packages from committed metadata:

   ```sh
   go run ./cmd/generate/ --skip-scan --metadata-dir ./metadata
   ```

2. **Reproducibility** — The committed metadata documents exactly what the generator saw from a specific SDK version. Diffs to `.gometa.json` files make Apple SDK changes visible in version control.

When a new SDK version is released, re-scan with:

```sh
go run ./cmd/generate/ --framework all --metadata-dir ./metadata
```

The `--metadata-dir ./metadata` flag is critical: without it the scanner writes to `.cache/bindmeta` (gitignored) and the committed metadata is not updated.
