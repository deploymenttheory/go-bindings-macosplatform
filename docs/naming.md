# Naming Standards

This document is the single source of truth for naming conventions in the codegen pipeline
(`internal/meta/`, `internal/scanner/`, `internal/codegen/`). Every rule here was agreed
explicitly; do not deviate without updating this document.

---

## 1. Boolean Fields — `Is*` prefix universally

Every boolean field on every struct uses the `Is*` prefix. No bare adjectives, no `Has*`,
`Needs*`, or `In*` prefixes.

```go
// Correct
IsNullable         bool
IsReadOnly         bool
IsClassMethod      bool
IsNSError          bool
IsNSStringOverloads bool

// Wrong — never use
Nullable           bool   // bare adjective
HasNSError         bool   // Has prefix banned
InClassMethod      bool   // In prefix banned
NeedsFormatPragma  bool   // Needs prefix banned
```

Affected renames from old code:

| Old | New |
| --- | --- |
| `Nullable` | `IsNullable` |
| `Readonly` | `IsReadOnly` |
| `Weak` | `IsWeak` |
| `Copy` | `IsCopy` |
| `NoEscape` | `IsNoescape` |
| `AlreadyRetained` | `IsAlreadyRetained` |
| `Unavailable` | `IsUnavailable` |
| `SwiftOnly` | `IsSwiftOnly` |
| `MainThreadRequired` | `IsMainThreadRequired` |
| `WarnUnused` | `IsWarnUnused` |
| `InClassMethod` (typemap.Context) | `IsClassMethod` |
| `NSStringOverloads` (Mapper) | `IsNSStringOverloads` |
| `HasNSError` | `IsNSError` |
| `HasMembers` | `IsNonEmpty` |

---

## 2. ObjC Method Parameters — `Param / Params / param`

An ObjC method or function parameter is always called `Param` (struct), `Params` (slice field),
and `param` (loop variable). The word "arg", "argument", or "parameter" is never used in code
(only in long-form comments when explaining ObjC concepts to readers).

```go
// Struct
type Param struct {
    Name      string
    ObjCType  string
    Direction string  // "", "out", "inout" — was: Modifier
    IsNullable bool
    IsNoescape bool
    IsBlock    bool
    BlockRef   string
}

// Field names
type Method struct {
    Params []Param  // was: Args []Arg
}

// Loop variable
for _, param := range method.Params {
    goType := mapper.GoType(param.ObjCType, ctx, imports)
}
```

Function renames:

- `ArgName()` in `naming.go` → `ParamName()`
- `resolveArgNames()` in emitters → `resolveParamNames()`
- `makeParam()` in scanner (was `buildArg()`)

---

## 3. Return Value Descriptor — `ReturnType` struct

The struct describing what an ObjC method returns is `meta.ReturnType`. The field that holds it
on `Method` and `Function` remains named `Return` (field name distinct from type name).

```go
type ReturnType struct {
    ObjCType          string
    IsNullable        bool
    IsInstancetype    bool
    IsAlreadyRetained bool
    IsGeneric         bool
    GenericParamName  string
}

type Method struct {
    Return ReturnType  // field name = Return, type name = ReturnType
}

// Local variable for the return descriptor
retType := method.Return
```

---

## 4. Framework vs Package — full words, no abbreviations

| Context | Canonical term | Variable convention |
| --- | --- | --- |
| Apple SDK framework (metadata, scanner) | `framework` | `framework *meta.FrameworkMeta` |
| Emitted Go package name (lowercase) | `package` | `packageName string` |
| Emitted Go package import path | `package` | `packagePath string` |
| Go import alias in generated code | `package` | `packageAlias string` |

**Banned variable names:** `fm`, `fw`, `fwLower`, `pkg`, `f` (for a framework).

```go
// Correct
for _, framework := range reg.Frameworks {
    packageName := strings.ToLower(framework.Name)
    packagePath  := modulePath + "/" + packageName
}

// Wrong
for _, fm := range reg.Frameworks {   // fm banned
    pkg := strings.ToLower(fm.Name)   // pkg banned
}
```

The receiver variable for `*meta.FrameworkMeta` is always `framework`:

```go
func EmitEnums(w io.Writer, framework *meta.FrameworkMeta, ...) error
```

---

## 5. ObjC Selector String — `selector` everywhere

The ObjC method selector string (e.g. `"initWithCString:encoding:"`) is always named `selector`
in variables and function parameters. Abbreviations `sel` and compound forms `methodSelector` are
banned.

```go
// Correct
func buildBridgeImplMethod(selector string, ...) bridgeImplMethodModel

// Wrong
func buildBridgeImplMethod(sel string, ...)            // sel banned
func buildBridgeImplMethod(methodSelector string, ...) // compound banned
```

---

## 6. Scanner Function Verbs — three-verb contract

Functions in `internal/scanner/` use a strict three-verb vocabulary:

| Verb | When to use | Examples |
| --- | --- | --- |
| `scan*` | Walk Clang AST children to identify or populate a meta struct | `scanClass()`, `scanMethod()`, `scanEnum()`, `scanProtocol()` |
| `parse*` | Operate on a raw string or YAML blob | `parseReturnType()`, `parseTBDClasses()`, `parseReplacedBy()` |
| `make*` | Assemble a meta struct from already-resolved pieces | `makeParam()`, `makeReturnType()` |

No other verbs (`extract*`, `build*`, `get*`) are used in `internal/scanner/`.

**`scan*` is exclusive to `internal/scanner/`.** Do not use it in `pipeline/`, `emit/`, or
`typemap/` — those packages use their own verb vocabulary below.

---

## 6a. Pipeline Helper Verbs

Private helpers in `internal/codegen/pipeline/` use this vocabulary:

| Verb | When to use | Examples |
| --- | --- | --- |
| `resolve*` | Derive a value from the registry or metadata | `resolveBlockedImports()` |
| `detect*` | Identify a problem (cycle, conflict) | `detectImportCycle()` |
| `locate*` | Find a file or path on disk | `locateSwiftInterface()` |
| `select*` | Choose among candidates | `selectBestArch()` |
| `lookup*` | Query a map or index | `lookupStruct()`, `lookupProtocol()` |
| `forEach*` | Iterate and apply a function to each element | `forEachFramework()` |
| `sort*By*` | Order by a named criterion | `sortFrameworksByDependency()` |
| `emit*` | Generate output for one unit (private mirror of public Emit*) | `emitFramework()`, `emitErgonomic()` |
| `index*` | Populate an index map as a side effect | `indexProtocolProxiesFromMethods()` |
| `build*` | Construct a data structure (no I/O) | `buildProtocolProxyIndex()` |
| `record*` | Write a side effect into a map or slice | `recordOpinionatedImports()` |

---

## 7. Emitter Function Naming — three-tier contract

All code in `internal/codegen/emit/` follows a three-tier naming convention:

| Tier | Visibility | Verb | Meaning |
| --- | --- | --- | --- |
| Entry-point | Exported | `Emit*` | Orchestrates writing one construct type for a framework |
| I/O helper | Unexported | `write*` | Writes bytes/strings to an `io.Writer` |
| Model constructor | Unexported | `build*` | Constructs a `*Model` template-data struct (no I/O) |

```go
// Entry-point (exported)
func EmitEnums(w io.Writer, framework *meta.FrameworkMeta, ...) error

// I/O helper (unexported)
func writeEnumMember(w io.Writer, member meta.EnumMember, ...) error

// Model constructor (unexported)
func buildEnumModel(e *meta.Enum, ...) enumModel
```

Bare noun entry-points like `Enums()`, `Structs()`, `Classes()` are renamed to `Emit*` form.

---

## 8. Template-Data Structs — `*Model` suffix

All structs that exist solely to feed a Go template (defined in `emit/raw/model.go`) use the
`*Model` suffix.

```go
type EnumMemberModel    struct { ... }
type EnumModel          struct { ... }
type BridgeHeaderModel  struct { ... }
type BlockSignatureModel struct { ... }  // was: BlockSignature
type MethodSigModel     struct { ... }  // was: MethodSig
```

Domain structs in `internal/meta/` do **not** use this suffix.

---

## 9. Registry Lookup Maps — `*Index` suffix

All lookup maps on `pipeline.Registry` use the `*Index` suffix to signal "pre-built fast-lookup
index derived from framework metadata". The iteration slice `Frameworks` is not an index.

```go
type Registry struct {
    Frameworks []*meta.FrameworkMeta  // source data, not an index

    ClassNameIndex     map[string]bool
    ClassIndex         map[string]meta.Class
    OwnerIndex         map[string]string
    EnumIndex          map[string]*meta.Enum
    ProtocolIndex      map[string]*meta.Protocol
    StructIndex        map[string]*meta.Struct
    TypedefIndex       map[string]string
    TypedefOwnerIndex  map[string]string
    CFTypeIndex        map[string]string
    GenericParamIndex  map[string][]string
    ProtocolProxyIndex map[string]bool
    EnumGoTypeIndex    map[string]string
}
```

Old-to-new map for the rename:

| Old | New |
| --- | --- |
| `KnownClasses` (bool set) | `ClassNameIndex` |
| `AllClasses` | `ClassIndex` |
| `FrameworkOwner` | `OwnerIndex` |
| `KnownEnums` | `EnumIndex` |
| `KnownProtocols` | `ProtocolIndex` |
| `KnownStructs` | `StructIndex` |
| `KnownTypedefs` | `TypedefIndex` |
| `KnownTypedefOwner` | `TypedefOwnerIndex` |
| `KnownFrameworkCFTypes` | `CFTypeIndex` |
| `KnownGenericParams` | `GenericParamIndex` |
| `KnownProtocolProxies` | `ProtocolProxyIndex` |
| `KnownEnumGoTypes` | `EnumGoTypeIndex` |

---

## 10. typemap.Context — inputs only; ImportSet for output

`typemap.Context` holds only caller-provided inputs. The import side-effect is captured in a
separate `typemap.ImportSet` passed explicitly to every resolution function.

```go
// inputs only
type Context struct {
    Framework     string
    KnownClasses  map[string]bool
    GenericParams []string
    IsReturn      bool
    IsClassMethod bool
}

// output: collected as a side-effect of type resolution
type ImportSet map[string]string  // package alias → import path

// function signatures
func (m *Mapper) GoType(qt string, ctx Context, imports ImportSet) string
func (m *Mapper) GoReturnType(qt string, ctx Context, imports ImportSet) string
```

Callers create an `ImportSet`, pass it in, then read the populated map:

```go
ctx     := typemap.Context{Framework: "foundation", ...}
imports := make(typemap.ImportSet)
goType  := mapper.GoType(qt, ctx, imports)
// imports is now populated with any cross-framework dependencies
```

---

## 11. Loop Variables — full descriptive word

Loop variables always use the full concept name. Single-letter abbreviations and short-forms
(`cls`, `meth`, `p`, `m`, `c`) are banned.

| Concept | Variable | Nested alternative |
| --- | --- | --- |
| ObjC class | `class` | `parentClass` |
| ObjC method | `method` | `parentMethod` |
| ObjC param | `param` | — |
| ObjC protocol | `protocol` | — |
| ObjC enum | `enum` | — |
| ObjC struct | `structDef` | — |
| Apple framework | `framework` | `parentFramework` |
| Enum member | `member` | — |

```go
// Correct
for _, class := range framework.Classes {
    for _, method := range class.Methods {
        for _, param := range method.Params { ... }
    }
}

// Wrong
for _, cls := range fm.Classes {     // cls and fm banned
    for _, m := range cls.Methods {} // m banned
}
```

---

## 12. Intermediate Buffers and Writers

| Role | Variable name |
| --- | --- |
| `io.Writer` function parameter | `w` |
| Primary `bytes.Buffer` for Go source | `buf` |
| `strings.Builder` for a string fragment | `sb` |
| Named sub-buffer (e.g. bridge header) | `headerBuf`, `implBuf` |

---

## 13. Bridge Terminology

"Bridge" has multiple meanings in this codebase. Use precise qualifiers:

| Meaning | Canonical phrase |
| --- | --- |
| The `.h`/`.m` file pair per framework | "C bridge layer" or "bridge file" |
| A C trampoline function | "bridge function" |
| ObjC ↔ Go type adaptation | "type mapping" — never "type bridging" |
| C symbol names resolved for a class | "bridge symbol names" |

`BridgeFuncName()` in `naming.go` is not renamed — it is unambiguous in its context.

---

## 14. Generated C Symbol Names — `goBridge_` prefix family

All C symbols generated into bridge files use the `goBridge_` prefix so that a
reader — including someone debugging a crash or stack trace — can immediately
identify the layer, mechanism, and target without prior knowledge of the
codebase. The format is:

```
goBridge_{Mechanism}_{Target}_{operation}
```

Segments use **camelCase** for the operation verb. Underscores separate the
three logical segments. The `Target` segment preserves the ObjC class or
protocol name verbatim (PascalCase as declared in the SDK).

### 14a. IMP trampoline getters

These functions return a C function pointer (`IMP`) for a specific ObjC method
signature, for use with `class_addMethod`. The `Mechanism` is `Trampoline`, the
role is `MethodFn`, and the signature shape follows as a type-list suffix.

```c
goBridge_Trampoline_MethodFn_void
goBridge_Trampoline_MethodFn_void_ptr_ptr
goBridge_Trampoline_MethodFn_bool_ptr_uint64
// pattern: goBridge_Trampoline_MethodFn_{returnType[_paramType...]}
```

The type-list suffix encodes the ObjC method return type then each parameter
type, separated by `_`. This encoding is stable and readable in debugger output.

### 14b. Callback bind / lookup

These functions store and retrieve the Go callback key associated with an ObjC
object and selector. They live in `internal/callbacks/callbacks_abi.h/.m` and
are hand-crafted (not generated).

```c
goBridge_Callback_Bind(obj, selName, key)   // stores key on obj for selector
goBridge_Callback_Lookup(obj, sel)           // retrieves key for dispatch
```

The internal ObjC associated-object class is `GoBridgeCallbackKey`
(PascalCase, no underscores — ObjC class name convention).

### 14c. Dynamic subclass factory functions

Generated once per eligible ObjC class (classes that appear in `SuperclassIndex`
and have at least one framework-specific overridable method). Three functions
per class:

```c
goBridge_Sub_{ClassName}_createClass()                              // allocates the dynamic class pair
goBridge_Sub_{ClassName}_addMethod(cls, sel, imp, enc)              // adds one method to the class
goBridge_Sub_{ClassName}_registerInit(cls)                          // registers pair and returns +1 instance
```

Example for `ABRecord`:
```c
goBridge_Sub_ABRecord_createClass
goBridge_Sub_ABRecord_addMethod
goBridge_Sub_ABRecord_registerInit
```

### 14d. Protocol callback factory functions

Generated once per eligible ObjC protocol (protocols with IMP-safe methods).
Same three-operation shape as subclass factories but with `Proto` as the
mechanism segment:

```c
goBridge_Proto_{ProtoName}_createClass()
goBridge_Proto_{ProtoName}_addMethod(cls, sel, imp, enc)
goBridge_Proto_{ProtoName}_registerInit(cls)
```

Example for `VZVirtualMachineDelegate`:
```c
goBridge_Proto_VZVirtualMachineDelegate_createClass
goBridge_Proto_VZVirtualMachineDelegate_addMethod
goBridge_Proto_VZVirtualMachineDelegate_registerInit
```

### 14e. Generated Go file names and types for the subclass/protocol layer

| Concept | File suffix | Go struct | Go factory |
| --- | --- | --- | --- |
| ObjC subclass from Go | `{Class}_subclass.go` | `{Class}Overrides` | `New{Class}Subclass` |
| ObjC protocol impl from Go | `{Proto}_protocol_callback.go` | `{Proto}Callbacks` | `New{Proto}ProtocolCallback` |

The `{Proto}Callbacks` struct holds the Go callback functions. The factory
parameter is named `fns` to avoid collision with the `callbacks` runtime package
import.

---

## Quick Reference Card

```text
Booleans          Is* always (IsNullable, IsReadOnly, IsClassMethod)
ObjC param        Param/Params/param  (never Arg/Args/arg)
Return descriptor ReturnType (struct), Return (field), retType (var)
Framework var     framework (never fm/fw/f)
Go package var    packageName / packagePath / packageAlias (never pkg)
Selector string   selector (never sel / methodSelector)
Scanner verbs     scan* | parse* | make*
Emitter tiers     Emit* (exported) | write* (I/O) | build* (model ctor)
Template structs  *Model suffix
Registry maps     *Index suffix
Loop vars         full word: class, method, param, protocol, framework
Buffers           w (writer param) | buf (bytes.Buffer) | sb (strings.Builder)
C symbols         goBridge_{Mechanism}_{Target}_{operation} (camelCase op)
IMP getters       goBridge_Trampoline_MethodFn_{sig}
Callback ABI      goBridge_Callback_Bind / goBridge_Callback_Lookup
Subclass C        goBridge_Sub_{Class}_createClass/addMethod/registerInit
Protocol C        goBridge_Proto_{Proto}_createClass/addMethod/registerInit
```
