# Emittance Pipeline Refactor — idiomatic bindings as a real compiler

Status: **approved, not yet started.** This document is the authoritative spec; it is
detailed enough to execute without re-deriving decisions. Read §2 (principles) first — every
later decision cites the principle it serves.

---

## 1. Context & problem

The idiomatic emitter (`internal/codegen/frameworks/emit/idiomatic/`) produces correct,
hermetic output, but it is a *fragile pipeline*, not a tool. Evidence from the current code:

- **Emission is half templates, half string-building.** ~90% of output runs through
  `templates/*.tmpl`, but method bodies (`plainMethodBody`, `plainMethodBodyWithOuts` in
  `idiomatic.go`), value structs (`structs.go`), error sentinels (`errors.go`), and all
  C-function wrappers (`functions.go`) are assembled with `fmt.Fprintf`/`bytes.Buffer`.
- **GATHER and RENDER are interleaved.** `emitClassFile` builds per-template *fragment* view
  structs (`classHeaderView`, `plainMethodView`, …) and renders them inline. There is no single
  object that represents "the Virtualization package we are about to emit."
- **Package-level mutable state** keyed by `*meta.FrameworkMeta`: `referencedEnumsCache`,
  `ownEnumNameCache`, `ownTypeNameCache`, `localValueStructNamesCache`, `classPrefixCache`. The
  worst, `referencedEnums(fw)`, is **mutated as a side effect of rendering class files** and
  read later by `emitEnums` — emission is order-dependent.
- **Imports are recovered by scanning rendered text** (`usedImports` greps the body for
  `alias.`), instead of being computed from the types the model references.

Bolting "readability" onto this would add heuristics to a brittle base. Instead we adopt the
compiler-style pipeline the maintainer specified:

```text
clang AST  ──scan──▶  <framework>.gometa.json     (exists)
                      appledocs.json              (exists; scripts/tools/appledeveloperdocs)
                          │
                          ▼
                    GATHER ─▶ one complete, immutable view.Framework
                              (hierarchy, docs, links, providers, members — all resolved)
                          │
                          ▼
                    RENDER ─▶ every .go file produced through .tmpl files only
```

Hard invariants (already true; re-checked after every migration step):
`bindings/frameworks` is never modified; no idiomatic file imports `bindings/frameworks`
(hermetic gate); the full 244-framework regen builds with **0 failing packages**.

---

## 2. Principles charter — the Zen of Go, made concrete

Source: <https://dave.cheney.net/2020/02/23/the-zen-of-go>. Each proverb below is restated as a
binding rule for **(T)** the emitter (tool) and **(O)** the emitted bindings (output). Later
sections reference these as P1–P11.

- **P1 — Each package fulfils a single purpose.**
  - (T) Three packages, each with one job: `view` (data only), `gather` (meta → view), `render`
    (view → files). `view` imports nothing from `gather`/`render`; `render` contains no
    resolution logic; `gather` produces no strings of Go syntax.
  - (O) One macOS framework ⇒ one Go package; support concerns (`obj`, `rt`, `errkit`,
    `objref`) stay in their own single-purpose packages.
- **P2 — Handle errors explicitly.**
  - (T) `gather.Framework` returns `(view.Framework, error)`; unresolved types are reported as
    structured diagnostics, never silently dropped without a recorded reason. `render` returns
    errors for I/O; a template execution failure is a programmer error and panics (caught by
    tests).
  - (O) Every fallible call returns `error` backed by `*errkit.Error`; NSError → error,
    sentinels via `errkit.New`; `BOOL+NSError` collapses to `error` (done). No in-band sentinels.
- **P3 — Return early rather than nesting deeply.**
  - (T) Gather resolvers return `(_, ok=false)` / `(_, err)` at the top on unhandled shapes;
    no deep `else` ladders.
  - (O) `method_body.tmpl` emits guard/error checks as early returns (`if _nsErr != 0 { return … }`)
    before the success path — never nested.
- **P4 — Leave concurrency to the caller.**
  - (O) Async completion-handler methods are surfaced as **blocking** `Foo(ctx context.Context) error`;
    the caller decides whether to run them in a goroutine. The binding never spawns goroutines.
- **P5 — Before you launch a goroutine, know when it will stop.**
  - (O) The async wrapper uses a **buffered** `chan, 1` and `select { case <-ctx.Done(): }`, so
    the completion block can always send and nothing leaks (the existing pattern; preserved and
    rendered from the IR, not string-built).
- **P6 — Avoid package-level state.**
  - (T) **Delete all five `map[*meta.FrameworkMeta]…` caches and the side-effecting
    `referencedEnums`.** Per-framework derived data (prefix, own-type sets, abstract bases,
    referenced enums, emittable structs, imports) lives in a `gather.Context` value created at
    the start of `gather.Framework` and discarded when it returns. `render` is a pure function
    of `view.Framework` with zero globals (template set is read-only, built once via `embed`).
  - (O) Generated packages keep only the unavoidable lazy `sync.Once` dlopen bootstrap and
    `objc.RegisterName` caches, which are immutable-after-init, not shared mutable state.
- **P7 — Simplicity matters.**
  - (T) The IR is the single source of truth; templates do iteration and substitution only (no
    type decisions). One resolver per construct; no string concatenation of Go syntax in gather.
  - (O) Idiomatic types only (`string`, `int`, `[]T`, `error`, wrapper structs); embedded
    hierarchy instead of provider plumbing; one obvious way to set a property.
- **P8 — Write tests to lock in the behaviour of your package's API.**
  - (T) Golden tests: `gather` → expected `view.Framework` for fixtures (the VZBootLoader
    hierarchy); `render` → expected `.go` golden files; a representative `method_body` golden
    per dispatch kind. Architecture gates (no `fmt.Fprintf` of Go in gather/render; no
    package-level mutable maps) enforced by a test that greps the source.
  - (O) Curated `example_test.go` per top framework double as compile/smoke tests; a negative
    `// build-fail` test proves sealed provider interfaces reject non-members.
- **P9 — If you think it's slow, first prove it with a benchmark.**
  - (T) No premature optimisation in gather/render; if regen time regresses, add a benchmark
    over the 244-framework gather before optimising. (No speculative concurrency in the tool.)
- **P10 — Moderation is a virtue.**
  - (O) Do not emit meaningless surface: no `New<AbstractBase>()`, no marker interfaces that
    every object satisfies, no `Set<X>`+`With<X>` duplication for polymorphic properties, no
    doc links to unrelated types. Generate what a user needs, nothing more.
- **P11 — Maintainability counts.**
  - (T) Adding a framework or a new construct means adding a resolver + a template, not editing
    a 3,000-line function. The IR documents intent.
  - (O) godoc reads like a manual: package overview, type index, hierarchy, cross-links, and
    runnable examples — all regenerated from metadata.

---

## 2b. Effective Go charter

Source: <https://go.dev/doc/effective_go>. Restated as binding rules for **(T)** the emitter and
**(O)** the emitted bindings; referenced later as E1–E14. These complement (do not contradict)
P1–P11.

- **E1 — gofmt.** "reads a Go program and emits the source in a standard style." (T) `render`
  feeds output through `emit.WriteGoFile` (gofmt) — templates need not be perfectly aligned.
  (O) every file is gofmt-clean.
- **E2 — Doc comments start with the identifier name.** "Comments that appear before top-level
  declarations … document the declaration itself." (T) every exported `view`/`gather`/`render`
  symbol; (O) every generated exported symbol — `// BootLoader is …`, `// NewMacOSBootLoader …`.
  Enforced by a gate (§8).
- **E3 — Package names: short, lowercase, no underscores, no stutter.** (T) `view`/`gather`/
  `render`; the type is `view.Framework`, never `view.ViewFramework`. (O) framework package
  names already lower-cased by `naming.PackageName`.
- **E4 — Getters have no `Get` prefix; setter is `SetX`.** "the getter method should be called
  `Owner`, not `GetOwner`." (O) property getters render as bare MixedCaps (`Count()`, `State()`);
  settable properties get `SetX` (+ fluent `WithX`). ObjC selectors literally named `getX:`
  (out-parameter fillers) keep their name — they are not field getters.
- **E5 — Interface naming.** "one-method interfaces are named by the method name plus an -er
  suffix." (T) behavioural single-method interfaces use `-er` (e.g. a render output sink). (O)
  the domain interfaces are **sum/marker** types, not single-method behavioural agents, so
  `<Base>Provider` / `<Type>able` is a **deliberate, documented divergence** — they name a role
  ("can be used as a boot loader"), not one method. The spec records this so it is a choice,
  not an accident.
- **E6 — MixedCaps, never underscores.** (O) all identifiers; de-prefixing preserves MixedCaps.
- **E7 — Multiple return values; named result parameters *where they clarify*.** "The names …
  can make code shorter and clearer: they're documentation." (O) methods with lifted
  out-parameters or an async typed result render **named returns**, e.g.
  `func (x *Foo) PropertyList(...) (plist obj.Object, format PropertyListFormat, err error)`;
  single-value returns stay unnamed (Effective Go: only when they clarify). This forces an IR
  delta (§2c).
- **E8 — Return early, avoid `else`, comma-ok, type switch.** "the unnecessary `else` is
  omitted." (T) gather resolvers return early on unhandled shapes. (O) `method_body.tmpl` emits
  guards/error checks as early returns; `obj.As` is the comma-ok narrowing.
- **E9 — Composite literals over `new`; useful zero values.** "taking the address of a composite
  literal allocates a fresh instance." (O) `FromID`/adopt use composite literals
  (`&MacOSBootLoader{BootLoader: BootLoader{Handle: objref.Wrap(id)}}`), not `x := &T{}; x.Handle=…`;
  value structs (`CGRect{}`) are usable at their zero value.
- **E10 — Pointer vs value receivers, consistent.** (O) wrappers use pointer receivers
  throughout; value structs carry no methods (plain data).
- **E11 — Compile-time interface checks.** "a global declaration using the blank identifier …
  `var _ json.Marshaler = (*RawMessage)(nil)`." (O) emit `var _ <Iface> = (*T)(nil)` for **both**
  the `<Type>able` mock interface **and every provider the type seals** — guaranteeing the sealed
  provider (WS2) is actually satisfied.
- **E12 — Error strings: lowercase, contextual, package-prefixed, no trailing punctuation;
  panic only for programmer error.** "error strings should identify their origin … 'image:
  unknown format'." (T) `fmt.Errorf("gather %s: %w", fw, err)`. (O) `errkit` messages stay
  contextual; panics are reserved for misuse (`errkit.CheckIndex` → `*errkit.Exception`),
  aligning with P2/P3.
- **E13 — Stringer.** "`func (t *T) String() string`." (O) enums already have `String()`;
  **add `String()` on framework-root wrappers** returning `-description` (promoted to all
  subclasses) so `fmt` prints something useful. (Deliberate balance of E13 vs P10: one method,
  high fmt value, zero per-subclass cost via promotion.)
- **E14 — Accept interfaces, return concrete types.** (O) setters accept the sealed provider
  interface; constructors return the concrete `*Wrapper`. (T) `gather.Framework` returns the
  concrete `view.Framework`; `render` accepts an output-sink interface for testability.

## 2c. Refinements forced by the Effective Go charter (IR/template deltas)

These are additive changes to §4–§6, listed here so they are not missed:

1. **Named returns (E7).** `view.Result` gains `Name string`; `view.OutParam` gains `Name`
   (the godoc-friendly return name, distinct from the internal `Local`). `view.Method` carries
   `Returns []NamedReturn` (the ordered named-return clause) when there is more than one return
   value or an out-parameter; `render` builds the signature `(...)` from it and
   `method_body.tmpl` assigns the locals to those names. Single-value methods keep an unnamed
   return.
2. **Composite-literal construction (E9).** The `class.tmpl` `FromID`/adopt blocks use a single
   composite literal threading the embedded base chain, not field assignment.
3. **Interface assertions (E11).** `render` emits `var _ I = (*T)(nil)` for the mock interface
   and for each `Class.Providers[i].Iface`.
4. **`String()` on roots (E13).** `Class.EmbedsRoot` ⇒ `class.tmpl` also emits
   `func (x *T) String() string { return rt.Description(objref.IDOf(x)) }`.
5. **Doc comments (E2).** Every `DocBlock` render starts with the symbol name; the `comment`
   FuncMap prepends it when the Apple prose does not already.
6. **Error-string style (E12).** gather/render error helpers are package-prefixed and lowercase;
   a gate checks no error string is capitalised or ends in punctuation.

## 2d. Principle tensions & resolutions

Where Zen and Effective Go pull in different directions, the ruling is recorded so it is a
decision, not drift:

- **T1 — Stringer (E13) vs Moderation (P10).** `String()` is emitted **only on framework-root
  wrappers** (one method, promoted to all subclasses); not duplicated per type, not added to
  value structs, not added where an enum already has one. Cost: one method per package root.
- **T2 — Named returns (E7) vs Simplicity (P7).** Named returns **only** when there is more than
  one return value or a lifted out-parameter; single-value methods stay unnamed.
- **T3 — `-er` naming (E5) vs domain `Provider`/`able`.** The emitted sum/marker interfaces name
  a *role* ("usable as a boot loader"), not a single behaviour, so they keep `<Base>Provider` /
  `<Type>able`. The **generator's own** single-method interfaces (e.g. the render sink) use `-er`.
- **T4 — Accept interfaces / return concrete (E14) vs emitting DI interfaces.** Constructors
  return the concrete `*Wrapper`; setters *accept* the sealed provider; the `<Type>able` mock
  interface exists for DI/testing but is never a return type. No conflict.
- **T5 — Avoid package state (P6) vs init-time singletons.** The `embed`-loaded template set and
  the generated `sync.Once` dlopen are **immutable-after-init**, which Go idiom permits; the ban
  is on *mutable* shared state (the 5 caches, the side-effecting `referencedEnums`).
- **T6 — DRY generator vs dumb templates (P7).** Resolution logic is centralised in `gather`
  (DRY); templates never duplicate it — they iterate and substitute only.
- **T7 — Doc richness (P11) vs Moderation (P10).** Links are limited to *direct* relationships
  (subclasses, the consuming setter, the embedded base, provider implementers). No transitive
  link spam.
- **T8 — Nominal sealing vs Go structural typing.** We deliberately add a nominal seal
  (`isBootLoader()`) so `WithBootLoader` accepts only real boot loaders — buying type safety and
  discoverability at the cost of one unexported marker method on the base.

## 2e. Principle → enforcement matrix

Each principle maps to a concrete mechanism, so "idiomatic" is testable, not aspirational.

| Principle | Enforced by (mechanism) | Checked by |
|---|---|---|
| P1 single purpose | `view`/`gather`/`render` split; `render` imports no `meta`/`typemap` | import-graph gate (§8.2) |
| P6 no package state | `gather.Context` holds all derived data; no `var … map[*meta.FrameworkMeta]` | source-scan gate (§8.2) |
| P7 simplicity | logic only in `gather`; templates iterate/substitute | no-`fmt.Fprintf`-of-Go gate; golden tests |
| P2/E12 errors | `error` returns + `errkit`; package-prefixed lowercase msgs | error-string gate (§8.2b) |
| P3/E8 return early | `method_body.tmpl` early-return structure | golden body tests |
| P4/P5 concurrency | `rt.Await` blocking + buffered chan + `ctx` | async golden test |
| E1 gofmt | `emit.WriteGoFile` | `gofmt -l` empty (§8.2b) |
| E2 doc comments | `comment` FuncMap prepends the name | ast doc-comment gate (§8.2b) |
| E4 getters/setters | naming in gather; `setter.tmpl` `Set`/`With` | golden tests |
| E7 named returns | `Method.Returns []NamedReturn`; `returns` FuncMap | ast named-return gate (§8.2b) |
| E11 interface checks | `render` emits `var _ I = (*T)(nil)` for mock + providers | ast assertion gate (§8.2b) |
| E13 Stringer | `class.tmpl` on `EmbedsRoot`; `enum.tmpl` | golden tests |
| E14 accept iface/return concrete | sealed providers + concrete `New*` | negative compile test (§8.5) |
| WS2 hierarchy | `Class.Base` embedding | VZBootLoader golden |
| WS3 prune | `Class.IsAbstract` ⇒ no ctors | golden (no `NewBootLoader`) |

---

## 3. Target architecture

### 3.1 Package layout (new)

```text
internal/codegen/frameworks/emit/idiomatic/
├── view/                      package view  — the IR; pure data, no behaviour, no emitter imports
│   ├── framework.go           Framework, Package, ImportSet, DocBlock, TypeRef
│   ├── class.go               Class, Ctor, Setter, MockIface
│   ├── method.go              Method, Param
│   ├── dispatch.go            Dispatch, Arg, Result, OutParam, Guard, ResultKind
│   ├── member.go              Enum, EnumMember, Struct, Field, Constant, Func, ErrorSentinel
│   └── example.go             Example
├── gather/                    package gather — meta+registry+docs → view.Framework (all logic)
│   ├── framework.go           Framework(meta, reg, docs) (view.Framework, error); the passes
│   ├── context.go             Context (per-framework derived data; replaces all package state)
│   ├── class.go               class + ctor + setter resolution; abstract detection; embedding
│   ├── method.go              method resolution → Method + Dispatch
│   ├── dispatch.go            decompose a selector call into a Dispatch (the marshaling brain)
│   ├── types.go               resolveParam / resolveReturn / resolveTypeRef (the type table)
│   ├── hierarchy.go           superclass walk, same-framework base, subclass lists
│   ├── providers.go           abstract bases → sealed Provider + implementer lists
│   ├── enums.go structs.go constants.go funcs.go sentinels.go
│   ├── docs.go                DocBlock assembly: clean prose + resolve [Type] links + usage
│   └── imports.go             ImportSet accumulation from TypeRefs
└── render/                    package render — view.Framework → files; templates only
    ├── render.go              Framework(v view.Framework, dir string) error
    ├── funcs.go               template FuncMap (pure formatting helpers only)
    └── templates/*.tmpl       one named template per construct + file + method_body
```

`view`, `gather`, `render` are siblings; the existing `idiomatic` package becomes a thin
wrapper (`EmitFrameworkWrappers` → `render.Framework(gather.Framework(...))`) during migration
and is deleted when empty. `naming`, `typemap`, `meta`, `appledocs`, `pipeline.Registry` are
reused unchanged.

### 3.2 Data flow

`GenerateIdiomatic` (pipeline) builds the `Registry` and merged `appledocs` once, then for each
framework: `v, err := gather.Framework(meta, reg, docs)` → `render.Framework(v, outDir)`.
Support packages: `gather.Support()` → `render.Support(dir)` (emitted from `support/*.txt` as
today, but driven by the same render entry point).

---

## 4. The IR (`view` package) — complete definitions

These are the exact types. Fields are populated entirely in `gather`; `render` only reads them.
(Serves P1, P7: render makes no decisions.)

```go
package view

type Framework struct {
    Package   Package
    Classes   []Class
    Providers []Provider     // sealed accepted-type interfaces (one per abstract base)
    Enums     []Enum
    Structs   []Struct
    Constants []Constant
    Funcs     []Func          // C-function wrappers
    Sentinels []ErrorSentinel
    Examples  []Example
}

type Package struct {
    Name       string         // "virtualization"
    ImportPath string
    BuildTag   string         // "//go:build darwin"
    Doc        PackageDoc     // rendered as doc.go
    Imports    ImportSet      // package-wide import set, computed from every TypeRef used
}

type PackageDoc struct {
    Summary    string         // Apple framework abstract (or a sensible default)
    Workflow   string         // a short, framework-specific "how to start" (entry types)
    Groups     []DocGroup     // type index grouped by abstract base
    Examples   []string       // names of Example funcs to point at
}
type DocGroup struct{ Title string; Types []TypeRef }   // e.g. "Boot loaders" → [MacOSBootLoader,…]

type DocBlock struct {
    Summary    string         // Apple "abstract", cleaned of HeaderDoc/Doxygen tags
    Discussion string         // Apple long prose if present (else "")
    Usage      string         // generated: "Construct with [NewX]; pass to [Config.WithY]."
    Links      []string       // resolved godoc references rendered as [Pkg.Type] in the comment
    AppleURL   string
    Deprecated string         // "since macOS X" if availability says so
}

// TypeRef is a fully-resolved reference to a Go type, carrying its import.
type TypeRef struct {
    GoName   string           // "MacOSBootLoader", "string", "[]*Foo", "corefoundation.CGRect"
    Import   *Import          // nil for builtins / same-package; set for cross-package
    DocLink  string           // godoc form for comments: "[MacOSBootLoader]" or "[corefoundation.CGRect]"
}

type Import struct{ Alias, Path string }     // Alias=="" means default (last path segment)
type ImportSet struct{ items []Import }       // ordered, deduped; Add(Import); List() []string

type Class struct {
    GoName, ObjCName string
    Doc        DocBlock
    IsAbstract bool                       // P10: no Ctors emitted
    Base       *TypeRef                   // same-framework base to EMBED; nil ⇒ embed objref.Handle
    EmbedsRoot bool                       // true when Base==nil (renders `objref.Handle` + obj.Object methods)
    Subclasses []TypeRef                  // concrete subclasses (for abstract-base docs)
    ConsumedBy []SetterRef                // "pass it to [Config.WithBootLoader]"
    Providers  []ProviderImpl             // sealed interfaces this class seals (marker method names)
    Ctors      []Ctor
    Setters    []Setter
    Methods    []Method
    Mock       MockIface
}
type SetterRef struct{ Owner, Method string }       // "VirtualMachineConfiguration", "WithBootLoader"
type ProviderImpl struct{ Iface, Marker string }    // "BootLoaderProvider", "isBootLoader"

type Ctor struct {
    GoName    string          // "NewMacOSBootLoaderWithURL"
    Doc       DocBlock
    Params    []Param
    Kind      CtorKind        // CtorNew | CtorAllocInit | CtorAllocInitError
    ClassName string          // ObjC class for _class(...)
    AdoptFn   string          // "macOSBootLoaderAdopt"
    InitSel   string          // selector for alloc/init kinds
    HasError  bool
    RetType   TypeRef         // *MacOSBootLoader
}
type CtorKind int
const ( CtorNew CtorKind = iota; CtorAllocInit; CtorAllocInitError )

type Setter struct {          // a settable property → SetX and/or WithX
    GoName   string           // "WithBootLoader" (and the Set variant flag)
    EmitSet  bool             // false to suppress redundant Set<X> for polymorphic props (P10)
    Selector string           // "setBootLoader:"
    Doc      DocBlock
    Param    Param
    Owner    string           // the wrapper type (for the With return)
}

type Param struct {
    GoName string             // safe Go identifier (reserved words / alias shadows handled)
    Type   TypeRef            // idiomatic Go type in the signature
    Arg    string             // marshal expression passed to objc.Send (e.g. "purego.NSString(s)")
    IsOut  bool               // value out-parameter → lifted to a return
}

type MockIface struct{ Name string; Methods []string }   // "BootLoaderable", method signature lines

type Provider struct {
    GoName       string       // "BootLoaderProvider"
    Base         string       // "BootLoader"
    Marker       string       // "isBootLoader" (unexported; sealed)
    Doc          DocBlock     // "Accepted by [Config.WithBootLoader]. Implemented by […]."
    Implementers []TypeRef
}
```

### 4.1 The Dispatch IR (the method body), fully specified

The single hardest thing to move out of string-building. The body of every instance/class
method, getter, setter, async, and slice getter reduces to this structure; `method_body.tmpl`
renders it with iteration only (P7).

```go
type Method struct {
    GoName, Selector string
    Doc      DocBlock
    Params   []Param          // signature params (out-params excluded; they're in Dispatch.Outs)
    Returns  []NamedReturn    // ordered return values; >1 value OR any out-param ⇒ render as
                              // NAMED returns (E7); a single value renders unnamed
    Dispatch Dispatch
}
type NamedReturn struct{ Name, GoType string }   // Name=="" ⇒ unnamed (single-value case)

type Dispatch struct {
    Style    DispatchStyle    // Plain | Async | Slice
    RecvExpr string           // "objref.IDOf(x)" (instance) | "objc.ID(_class(\"NSFoo\"))" (class func)
    Selector string           // "objectAtIndex:"
    Guards   []Guard          // pre-call checks, rendered first as early returns/panics
    SendType string           // T in objc.Send[T] — "objc.ID","bool","int", value-struct, …
    Args     []string         // marshaled arg expressions, in selector order (out-params use "unsafe.Pointer(&_outN)")
    Error    bool             // append unsafe.Pointer(&_nsErr); emit error check + nil
    Outs     []OutParam       // value out-params, lifted to returns
    Result   Result           // primary return conversion

    // Async only (Style==Async):
    Await      string         // "rt.Await" | "rt.AwaitValue"
    BlockArgs  []string        // objc.Block closure params (e.g. "_p0 objc.ID")
    BlockBody  []string        // statements inside the block (error/result marshal + done(...))

    // Slice only (Style==Slice):
    ElemConv string           // "func(_id objc.ID) *Foo { return FooFromID(_id) }"
    ElemType string           // "*Foo"
}
type DispatchStyle int
const ( Plain DispatchStyle = iota; Async; Slice )

type Guard struct{ Expr string }         // e.g. "errkit.CheckIndex(index, x.Count())"
type OutParam struct{ Local, Name, GoType, Zero string }   // Local="_out0", Name="format" (E7 named return)
type Result struct {
    Kind   ResultKind
    Name   string           // named-return identifier when part of a multi-value return (E7); else ""
    GoType string           // "" for void
    Wrap   string           // conversion template, one %s for the raw result: "obj.Wrap(%s)","purego.GoString(%s)","%s"
    Zero   string           // zero literal for the error path: "nil","\"\"","0","false"
}
type ResultKind int
const ( RVoid ResultKind = iota; RScalar; RBool; REnum; RString; RObject; RArray )
```

This captures every decision (marshaling, kinds, error/out handling) as **data**. See §6.3 for
the template that renders it.

### 4.2 Members

```go
type Enum struct{ GoName, GoType string; Doc DocBlock; Bitmask bool; Members []EnumMember; Default string }
type EnumMember struct{ GoName, Value string; Doc DocBlock; Zero bool }

type Struct struct{ GoName string; Doc DocBlock; Fields []Field }
type Field struct{ GoName, GoType string }          // value-struct fields: primitive or same-pkg struct only

type Constant struct{ GoName, ExternName string; Doc DocBlock; Kind ConstKind }  // CF ref / NSString / objc.ID
type Func struct{ GoName, CName string; Doc DocBlock; Params []Param; Dispatch CFuncDispatch }  // C-function wrapper
type ErrorSentinel struct{ GoName, Domain, Code string; Doc DocBlock }
type Example struct{ Name string; Body string }     // curated example_test.go funcs (hand-authored preserved)
```

---

## 5. Gather phase (`gather`) — the brain

`gather.Framework(meta, reg, docs) (view.Framework, error)` runs ordered passes over a single
`Context`. **No package-level state** (P6): the `Context` is created here and dies here.

```go
type Context struct {
    fw   *meta.FrameworkMeta
    reg  *pipeline.Registry
    docs *appledocs.Docs
    m    *typemap.Mapper

    // derived once, up front (was the 5 package caches):
    pkg        string                 // "virtualization"
    rawAlias   string                 // legacy alias used by typemap; internal
    prefix     string                 // de-prefix prefix ("VZ"); from detectClassPrefix
    classes    map[string]meta.Class  // own classes
    enumNames  map[string]bool        // own exported enum Go names
    typeNames  map[string]bool        // own exported type names
    structEmit map[string]bool        // emittable value structs (global set, shared via reg)
    abstract   map[string]string      // ObjC base → Go base name (subclass-derived)
    subclasses map[string][]string    // base → concrete subclasses
    setterOf   map[string]SetterRef   // provider-base → the Config setter that consumes it

    // accumulated DURING gather, frozen into the view at the end (not a global, not order-dependent):
    referenced map[string]bool        // enum Go names referenced by any resolved member
    imports    *view.ImportSet
}
```

### 5.1 Passes (ordered, deterministic — P3, P6)

1. **Index** own classes/enums/structs; compute `prefix`, `abstract`, `subclasses`,
   `setterOf` (walk every Config `With*` whose param is a provider). `structEmit` from the
   shared `reg.EmittableStructs` (already computed globally).
2. **Resolve classes**: for each class build `view.Class` (hierarchy/embedding §5.3, ctors §5.4,
   setters, methods §5.5, provider memberships, mock iface, docs §5.6). Resolving methods
   *records* referenced enums into `ctx.referenced` and types into `ctx.imports`.
3. **Resolve providers** (§5.7), **enums** (only those in `ctx.referenced` + `*ErrorCode`),
   **structs**, **constants**, **funcs**, **sentinels**.
4. **Package doc** (§5.6) and freeze `ctx.imports` into `Package.Imports`.
5. Return `view.Framework`. Unresolved members are dropped **with a recorded diagnostic** (P2),
   never silently.

### 5.2 Type resolution (`types.go`) — the table, as pure functions

`resolveParam(objcType) (view.Param, ok)` and `resolveReturn(objcType) (view.Result,
[]view.OutParam, hasErr, ok)` port today's `idiomaticArg`/`idiomaticRet`/`qualifyRaw` decisions
into pure functions returning IR. The mapping (unchanged semantics, now data):

| ObjC type | Go signature type | Marshal in / Wrap out |
|---|---|---|
| `NSString *` | `string` | in `purego.NSString(s)` / out `purego.GoString(%s)` |
| `NSURL *` (file) | `string` | in `rt.FileURL(s)` |
| `NSArray<E> *` | `[]E'` | in `purego.SliceToNSArray` / out `purego.NSArrayToSlice` |
| same-fw class `*` | `*Wrapper` | in `objref.IDOf(x)` / out `WrapperFromID(%s)` |
| cross-fw class `*` / `id` | `obj.Object` | in `objref.IDOf(x)` / out `obj.Wrap(%s)` |
| CF opaque ref (pointer) | `obj.Object` | as object |
| value struct (same fw) | `Struct` | by value |
| value struct (other fw) | `pkg.Struct` (+import) | by value |
| enum | de-prefixed enum type | passthrough (record referenced) |
| `NSUInteger`/`NSInteger` | `int` | passthrough |
| `T *` to scalar/enum | (lifted) `OutParam` | `unsafe.Pointer(&_outN)` |
| `NSError **` | (drops to `error`) | sets `Dispatch.Error` |
| block `^` | Go `func(...)` | `objc.NewBlock(adapter)` |

`resolveTypeRef` returns a `view.TypeRef` with its `Import` and `DocLink`, and **adds the import
to `ctx.imports`** (P6: imports computed here, never scanned — kills `usedImports`).

### 5.3 Hierarchy & embedding (`hierarchy.go`) — WS2

For class `C` with `meta.Class.Super == S`:
- if `S` is a class **in this framework** that is itself emitted ⇒ `Class.Base =
  TypeRef{GoName: deprefix(S)}`, `EmbedsRoot=false`. Subclass inherits base methods by promotion.
- else (framework root, e.g. super is Foundation `NSObject`) ⇒ `Base=nil`, `EmbedsRoot=true`
  (renders `objref.Handle` + the `Description/IsEqual/IsKind` methods).
Same-framework embedding follows the acyclic superclass DAG ⇒ no import cycles (P7).
`obj.Object` methods are emitted **only** on roots; subclasses promote them.
`Subclasses[base]` is the list of concrete subclasses (for docs).

### 5.4 Constructors (`class.go`) — WS3 (P10)

- **Abstract base** (in `ctx.abstract`) ⇒ `IsAbstract=true`, **no `Ctors`** (`New<Base>()` is
  meaningless).
- Concrete class ⇒ `CtorNew` for `+new`, `CtorAllocInit[Error]` for each designated/param init.
  Construction renders via the promoted handle: `x := &T{}; x.Handle = objref.Wrap(id);
  objref.Track(x)` (works through embedding).

### 5.5 Method resolution → Dispatch (`method.go`, `dispatch.go`)

`buildMethod` chooses `DispatchStyle` (Async if completion-handler; Slice if NSArray getter;
else Plain), resolves params/returns via §5.2, lifts value out-params, sets `Error`, fills
`Result`, and attaches `Guards` (e.g. bounds-checked `objectAtIndex:`). All branching lives
here; the output is the `Dispatch` data. The current `plainMethodBody`/`plainMethodBodyWithOuts`
/async/slice string builders are **deleted** — their logic becomes the Dispatch construction.

### 5.6 Docs & links (`docs.go`) — WS4 (P11)

`DocBlock` for a symbol = cleaned Apple `Summary`/`Discussion` (strip `@abstract`/`@discussion`/
`@see`/… via the existing `cleanDoc`) + generated `Usage` + resolved `Links`:
- abstract base: Usage = "Abstract base — don't construct it. Concrete types: [Sub…]. Pass one
  to [Owner.WithX]."; Links = subclasses + the consuming setter.
- concrete: Usage = "Construct with [NewX]; pass to [Owner.WithX]." + "Embeds [Base]."
- `PackageDoc`: Summary (framework abstract) + Workflow (the `*Configuration` entry type and its
  `With*` chain) + Groups (one per abstract base → its subclasses) + Examples.
Links render as godoc `[Type]` / `[Pkg.Type]` (cross-package gets the import too).

### 5.7 Providers (`providers.go`) — WS2 (P7, P10)

For each abstract base `B`: `view.Provider{GoName: B+"Provider", Base: B, Marker: "is"+B}` with
`Implementers` = subclasses. Render seals it: `type BProvider interface { objref.Object;
isB() }` and `func (x *B) isB() {}`. Every embedder of `B` promotes `isB` ⇒ only real `B`s
satisfy `BProvider`. `Config.WithB(BProvider)` now type-checks membership (the marker→sealed
upgrade is the type-safety win).

---

## 6. Render phase (`render`) — templates only

`render.Framework(v view.Framework, dir string) error` writes one `.go` per construct group
(classes, providers, enums, structs, constants, funcs, errors, doc.go) by executing the
template set against `v`. No `fmt.Fprintf` of Go syntax anywhere in `render` (P1, P7). Imports
come from `v.Package.Imports` / per-file subsets — never scanned.

### 6.1 Template set (composed; `text/template` with named sub-templates)

`templates/*.tmpl` embedded once (read-only; P6). A `file.tmpl` provides the scaffold and
includes named templates:

```text
file.tmpl            header + build tag + package + imports + {{block "body" .}}
class.tmpl           struct (embed Base|objref.Handle) + FromID/adopt + (root: obj.Object methods)
                     + {{range .Ctors}}{{template "ctor" .}} + setters + methods + mock + providers-impl
ctor.tmpl  setter.tmpl  method.tmpl  method_body.tmpl  (§6.3)
provider.tmpl        sealed interface + marker method + implementer doc
enum.tmpl struct.tmpl constants.tmpl funcs.tmpl errors.tmpl  docfile.tmpl  mock_interface.tmpl
```

`method.tmpl` emits the signature + doc and delegates the body to `method_body.tmpl`.

### 6.2 Template FuncMap (`funcs.go`)

Pure formatting only: `comment` (prose → `// ` block), `goList` (join), `sprintf`-style `wrap`
(apply a `%s` template), `args` (join Dispatch args). No type logic.

### 6.3 `method_body.tmpl` — the exact spec

Renders `view.Dispatch`. Logic is iteration/conditionals over pre-decided data (P7); it never
decides a type. Plain style:

```gotemplate
{{define "method_body"}}
{{- range .Guards}}{{.Expr}}
{{end -}}
{{- range .Outs}}var {{.Local}} {{.GoType}}
{{end -}}
{{- if .Error}}var _nsErr uintptr
{{end -}}
{{- if eq .Result.Kind 0 /*RVoid*/ -}}
{{call_send .}}
{{- if .Error}}
if _nsErr != 0 { return errkit.FromObjC(purego.NSErrorToError(objc.ID(_nsErr))) }
{{- range .Outs}}{{/* void+outs: return outs */}}{{end}}
return {{outs_and_nil .}}{{end}}
{{- else -}}
_r := {{call_send .}}
{{- if .Error}}
if _nsErr != 0 { return {{.Result.Zero}}{{outs_zero .}}, errkit.FromObjC(purego.NSErrorToError(objc.ID(_nsErr))) }
{{- end}}
return {{wrap .Result "_r"}}{{outs_values .}}{{if .Error}}, nil{{end}}
{{- end -}}
{{end}}
```

`call_send` (FuncMap) assembles `objc.Send[SendType](RecvExpr, objc.RegisterName("Selector"),
Args…[, unsafe.Pointer(&_nsErr)])` from the data — a pure string join, no decisions. Async and
Slice styles are separate `{{define}}` blocks rendering `rt.Await…`/`NSArrayToSlice` from the
same struct. Each style has a **golden test** (§8).

> **Construction form (resolves E9 vs P7).** Deep embedding chains make nested composite
> literals (`&A{B{C{Handle:…}}}`) ugly, so every `FromID`/adopt uses the promoted-field form
> `x := &T{}; x.Handle = objref.Wrap(…)` — uniform, depth-independent, simple (P7). This is the
> one place clarity (P7) is chosen over the composite-literal preference (E9), and it is recorded.

### 6.4 Template catalogue (complete)

Every template lives in `render/templates/`, is loaded once via `//go:embed *.tmpl`, and is
executed **only** against a `view` value (no `meta`/`typemap` reachable — P1). `render.Framework`
emits these files, computing each file's import subset from the `view` (never by scanning text):

| Output file | Top template | Body templates used |
|---|---|---|
| `doc.go` | `docfile` | — |
| `<pkg>_runtime_generated.go` | `runtime` | — |
| `<Class>_generated.go` | `header` + `class` | `ctor`,`setter`,`method`,`method_body`,`mock` |
| `<pkg>_providers_generated.go` | `header` + range `provider` | — |
| `<pkg>_enums_generated.go` | `header` + range `enum` | — |
| `<pkg>_structs_generated.go` | `header` + range `struct` | — |
| `<pkg>_constants_generated.go` | `header` + range `constant` | — |
| `<pkg>_functions_generated.go` | `header` + range `func` | `method_body` (cfunc variant) |
| `<pkg>_errors_generated.go` | `header` + range `sentinel` | — |

Each file is `header` (the shared scaffold) concatenated with its body templates, then run
through `emit.WriteGoFile` (gofmt — E1). The complete expected templates:

**`header.tmpl`** — input `headerView{Pkg string; Imports []string}`:

```gotemplate
{{define "header" -}}
// Code generated by go-bindings-codegen. DO NOT EDIT.

//go:build darwin

package {{.Pkg}}
{{if .Imports}}
import (
{{- range .Imports}}
	{{.}}
{{- end}}
)
{{end -}}
{{end}}
```

**`class.tmpl`** — input `view.Class`. Embeds base (WS2), emits root-only obj.Object+`String`
(E13), then members, sealed-marker methods, and the interface assertions (E11):

```gotemplate
{{define "class" -}}
{{comment .GoName .Doc}}
type {{.GoName}} struct {
{{- if .EmbedsRoot}}
	objref.Handle
{{- else}}
	{{.Base.GoName}}
{{- end}}
}

// {{.GoName}}FromID adopts an existing Objective-C object as a {{.GoName}} (nil for 0).
func {{.GoName}}FromID(id objc.ID) *{{.GoName}} {
	if id == 0 {
		return nil
	}
	x := &{{.GoName}}{}
	x.Handle = objref.Wrap(purego.Retain(id))
	objref.Track(x)
	return x
}

func {{.AdoptName}}(id objc.ID) *{{.GoName}} {
	if id == 0 {
		return nil
	}
	x := &{{.GoName}}{}
	x.Handle = objref.Wrap(id)
	objref.Track(x)
	return x
}
{{if .EmbedsRoot}}
func (x *{{.GoName}}) Description() string          { return rt.Description(objref.IDOf(x)) }
func (x *{{.GoName}}) IsEqual(other obj.Object) bool { return rt.IsEqual(objref.IDOf(x), objref.IDOf(other)) }
func (x *{{.GoName}}) IsKind(className string) bool  { return rt.IsKind(objref.IDOf(x), className) }
func (x *{{.GoName}}) String() string               { return rt.Description(objref.IDOf(x)) }
{{end}}
{{- range .Ctors}}{{template "ctor" .}}{{end}}
{{- range .Setters}}{{template "setter" .}}{{end}}
{{- range .Methods}}{{template "method" .}}{{end}}
{{- $T := .GoName}}{{range .Providers}}
func (x *{{$T}}) {{.Marker}}() {}
{{- end}}
{{template "mock" .Mock}}
{{range .Providers}}var _ {{.Iface}} = (*{{$T}})(nil)
{{end -}}
var _ {{.Mock.Name}} = (*{{.GoName}})(nil)
{{end}}
```

**`ctor.tmpl`** — input `view.Ctor`; no `Ctors` entries exist when `Class.IsAbstract` (WS3):

```gotemplate
{{define "ctor" -}}
{{comment .GoName .Doc}}
func {{.GoName}}({{params .Params}}){{returns .Returns}} {
{{- if eq .Kind 0 /*CtorNew*/}}
	return {{.AdoptFn}}(objc.Send[objc.ID](objc.ID(_class({{printf "%q" .ClassName}})), objc.RegisterName("new")))
{{- else}}
	_alloc := objc.Send[objc.ID](objc.ID(_class({{printf "%q" .ClassName}})), objc.RegisterName("alloc"))
{{- if .HasError}}
	var _nsErr uintptr
	_id := objc.Send[objc.ID](_alloc, objc.RegisterName({{printf "%q" .InitSel}}){{argsTail .}}, unsafe.Pointer(&_nsErr))
	if _nsErr != 0 {
		return nil, errkit.FromObjC(purego.NSErrorToError(objc.ID(_nsErr)))
	}
	return {{.AdoptFn}}(_id), nil
{{- else}}
	return {{.AdoptFn}}(objc.Send[objc.ID](_alloc, objc.RegisterName({{printf "%q" .InitSel}}){{argsTail .}}))
{{- end}}
{{- end}}
}
{{end}}
```

**`setter.tmpl`** — input `view.Setter`; `WithX` (fluent) and, unless suppressed (P10), `SetX`:

```gotemplate
{{define "setter" -}}
{{comment .GoName .Doc}}
func (x *{{.Owner}}) {{.GoName}}({{.Param.GoName}} {{.Param.Type.GoName}}) *{{.Owner}} {
	objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName({{printf "%q" .Selector}}), {{.Param.Arg}})
	return x
}
{{if .EmitSet}}
func (x *{{.Owner}}) Set{{trimWith .GoName}}({{.Param.GoName}} {{.Param.Type.GoName}}) {
	objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName({{printf "%q" .Selector}}), {{.Param.Arg}})
}
{{end -}}
{{end}}
```

**`method.tmpl`** — signature + doc; delegates the body to the right `method_body` variant:

```gotemplate
{{define "method" -}}
{{comment .GoName .Doc}}
func (x *{{.Dispatch.RecvOwner}}) {{.GoName}}({{params .Params}}){{returns .Returns}} {
{{- if eq .Dispatch.Style 1 /*Async*/}}{{template "method_body_async" .Dispatch}}
{{- else if eq .Dispatch.Style 2 /*Slice*/}}{{template "method_body_slice" .Dispatch}}
{{- else}}{{template "method_body" .Dispatch}}{{end}}
}
{{end}}
```

(Class/static functions use a `classfunc` variant: no receiver, `RecvExpr` is the class object,
same `method_body`.)

**`method_body.tmpl`** — the plain variant is in §6.3; the **async** and **slice** variants:

```gotemplate
{{define "method_body_async" -}}
	return {{.Await}}(ctx, func(done func({{.AwaitDoneSig}})) {
		_block := objc.NewBlock(func(_ objc.Block{{range .BlockArgs}}, {{.}}{{end}}) {
{{- range .BlockBody}}
			{{.}}
{{- end}}
		})
		{{send .}}
	})
{{end}}

{{define "method_body_slice" -}}
	_r := {{send .}}
	return purego.NSArrayToSlice(_r, {{.ElemConv}})
{{end}}
```

**`provider.tmpl`** — input `view.Provider`; the sealed interface (WS2) + implementer doc (WS4):

```gotemplate
{{define "provider" -}}
{{comment .GoName .Doc}}
type {{.GoName}} interface {
	objref.Object
	{{.Marker}}()
}
{{end}}
```

**`enum.tmpl`** — input `view.Enum`; typed const block + `String()` (E13):

```gotemplate
{{define "enum" -}}
{{comment .GoName .Doc}}
type {{.GoName}} {{.GoType}}

const (
{{- range .Members}}
	{{.GoName}} {{$.GoName}} = {{.Value}}
{{- end}}
)

func (e {{.GoName}}) String() string {
	switch e {
{{- range .Members}}
	case {{.GoName}}:
		return {{printf "%q" .GoName}}
{{- end}}
	default:
		return fmt.Sprintf({{printf "%q" .Default}}, {{.GoType}}(e))
	}
}
{{end}}
```

**`struct.tmpl`** — input `view.Struct`; value type, useful zero value (E9):

```gotemplate
{{define "struct" -}}
{{comment .GoName .Doc}}
type {{.GoName}} struct {
{{- range .Fields}}
	{{.GoName}} {{.GoType}}
{{- end}}
}
{{end}}
```

**`sentinel.tmpl`** — input `view.ErrorSentinel`:

```gotemplate
{{define "sentinel" -}}
{{comment .GoName .Doc}}
var {{.GoName}} = errkit.New({{printf "%q" .Domain}}, {{.Code}})
{{end}}
```

**`constant.tmpl`**, **`func.tmpl`**, **`mock.tmpl`**, **`docfile.tmpl`**, **`runtime.tmpl`** —
analogous one-construct templates: `func` reuses the cfunc `method_body` (RegisterLibFunc +
marshal); `mock` lists the type's own+promoted method signatures and names the interface
`<Type>able`; `docfile` renders `view.PackageDoc` (Summary, Workflow, `Groups` as a
`[Type]`-linked index, Examples); `runtime` is today's dlopen bootstrap, unchanged.

### 6.5 Template FuncMap (complete) — `render/funcs.go`

Pure, side-effect-free formatting helpers only (P1/P7); each is unit-tested:

| Func | Signature | Returns |
|---|---|---|
| `comment` | `comment(name string, d view.DocBlock) string` | godoc block whose first word is `name` (E2); appends Usage, resolved `[Type]` links, `Apple: <url>`, and a `Deprecated:` line when set |
| `params` | `params(ps []view.Param) string` | `name Type, name2 Type2` (omits `IsOut` params) |
| `returns` | `returns(rs []view.NamedReturn) string` | `""` / ` Type` / ` (name Type, …, err error)` — named iff `len>1` (E7) |
| `send` | `send(d view.Dispatch) string` | `objc.Send[SendType](RecvExpr, objc.RegisterName("Sel"), arg…[, unsafe.Pointer(&_nsErr)])` |
| `wrap` | `wrap(tmpl, expr string) string` | `fmt.Sprintf(tmpl, expr)` (apply the one `%s`) |
| `argsTail` | `argsTail(c view.Ctor) string` | `", arg, arg2"` (leading comma) or `""` |
| `trimWith` | `trimWith(s string) string` | strips a leading `With` for the `Set` name |
| `printf` | builtin | quoting via `%q` |

No FuncMap helper makes a type decision; all such decisions are already in the `view` values
(P7). A test asserts the FuncMap set is exactly this list, so logic can't creep into render.

### 6.6 Golden fixtures (lock the contract — P8 / E-gates)

Fixtures: `gather/testdata/` (input `.gometa.json` slices + appledocs) and `render/testdata/`
(expected `.go`). The locked set, chosen to exercise every branch:

- **Hierarchy/providers/docs:** the `VZBootLoader` family — abstract base (no ctor), three
  concrete embedders, sealed `BootLoaderProvider`, the `WithBootLoader` consumer, cross-links.
- **Dispatch kinds (one golden body each):** `count`→scalar; `objectAtIndex:`→object + bounds
  guard; `componentsJoinedByString:`→string; `componentsSeparatedByString:`→`[]string` (slice);
  `writeToURL:error:`→`error` (bool+NSError); `propertyListWithData:options:format:error:`→named
  multi-return with out-param; `startWithCompletionHandler:`→async `(ctx) error`.
- **Members:** a de-prefixed enum with `String()`; `CGRect` value struct (nested same-pkg
  structs); a `*ErrorCode`→sentinels; a CF-constant accessor; a C-function wrapper.
- **Edge cases (§9):** cross-framework value-struct return (`corefoundation.CGRect` import); a
  block param; an out-of-framework base (flat root); a de-prefix collision.

Golden tests run `gather`→compare `view.Framework` (struct equality) and `render`→compare bytes;
a `-update` flag regenerates goldens after an intentional change.

---

## 7. Migration sequence (incremental; gated)

In-place, one construct at a time. After **every** step, the gate must pass:
`go run ./cmd/generate/ idiomatic --framework all` then `go build ./opinionated/idiomatic/...`
⇒ 0 failing packages, 0 hermetic violations, `git diff --quiet bindings/frameworks`. Iterate on
**Virtualization** then **Foundation** before the global regen.

| Step | Work | Definition of done |
|---|---|---|
| 1 | Scaffold `view`/`gather`/`render`; port **enums + structs** end-to-end through the new path; delete their string-building | enums/structs identical output (byte-diff vs prior regen); golden tests pass; gate green |
| 2 | **Method bodies** → `method_body.tmpl` via `Dispatch`; delete `plainMethodBody*` and the async/slice string builders | identical method output; golden per dispatch kind; gate green |
| 3 | Port **C-funcs, errors, constants, providers, doc.go, file assembly**; replace `usedImports` with gathered `ImportSet` | no Go-syntax `fmt.Fprintf` left in emitter; imports identical; gate green |
| 4 | Delete the **5 package caches + `referencedEnums`**; move into `gather.Context` | architecture-gate test passes (no `map[*meta.FrameworkMeta]` vars); gate green |
| 5a | **WS2** embedding + sealed providers | VZBootLoader: `MacOSBootLoader struct{ BootLoader }`, sealed `BootLoaderProvider`; negative compile test; gate green |
| 5b | **WS3** prune abstract ctors / redundant setters | no `NewBootLoader()`; gate green |
| 5c | **WS4** docs: links, usage, provider implementers, `doc.go` overview/type-index | `go doc` reads as a manual; gate green |
| 5d | **WS5** curated `example_test.go` for Virtualization/Foundation/AppKit | `go test` of those packages passes |
| 6 | Zen pass + final validation (§2, §8) | all gates + golden + examples + go-doc smoke pass |

## 8. Verification

1. **Golden tests** (`gather`, `render`): expected `view.Framework` for the VZBootLoader fixture
   (Base/Subclasses/IsAbstract/Providers/links); expected rendered `.go`; one `method_body`
   golden per `ResultKind`/`DispatchStyle` (void, scalar, string, object, array, out-param,
   async, slice, bool+NSError).
2. **Architecture gates** (a Go test that scans source): zero `fmt.Fprintf`/`bytes.Buffer`
   producing Go syntax in `gather`/`render`; zero package-level mutable `map[*meta.FrameworkMeta]`
   (P6); `render` imports no `meta`/`typemap` (P1).
2b. **Effective Go conformance gates** (over the regenerated tree, via `go/ast`):
   `gofmt -l` is empty (E1); every exported decl has a doc comment whose first word is its name
   (E2); no exported identifier contains an underscore (E6); every concrete wrapper has a
   `var _ <Iface> = (*T)(nil)` for its mock and provider interfaces (E11); methods with >1
   return value or an out-parameter use named returns (E7); `go vet` and `golangci-lint run` are
   clean on the test-bed packages.
3. **Global gate** (§7).
4. **godoc smoke**: `go doc .../virtualization` and `.../virtualization.BootLoader`.
5. **Examples + type-safety**: `go test` the example packages; a `//go:build ignore` negative
   sample proving a non-bootloader is rejected by `WithBootLoader`.

## 9. Edge-case catalogue (must be handled in gather; each gets a fixture)

Generics instantiated as `objc.ID` (uniform flat wrapper); cross-framework base ⇒ flat root;
cross-framework value struct ⇒ owner-qualified `TypeRef`+import; CF integer-handle "…Ref" (not
an object) ⇒ scalar; OSStatus → `error`; value & scalar out-params; async typed-result vs
error-only; bool+NSError → `error`; block params/returns; abstract-base detection (no metadata
flag — derived from being a same-framework superclass); de-prefix collisions (fall back to
prefixed name; reserve enum names before funcs); param name shadowing (`safeParamName`); the
`objref.Handle` field vs a generated `Handle` method; hand-authored file preservation
(`scanHandAuthored`).

## 10. Risks & mitigations

- **Method-body templating** (highest risk): done early (step 2) behind golden tests; all
  decisions in the `Dispatch` IR so the template is mechanical.
- **Embedding regen touches ~every class**: incremental + global/hermetic gate each step;
  same-framework-only embedding (acyclic).
- **Generic `doc.go` quality varies across 244 frameworks**: solid generated default + curated
  hand-authored override allowed for top frameworks via `scanHandAuthored`.
- **Behaviour drift during port**: steps 1–4 require **byte-identical** regen output vs the
  pre-refactor tree (the refactor changes structure, not output); only steps 5a–5d intentionally
  change output, each behind its own gate.

---

# Appendices — full specification (no open decisions)

## Appendix A — Complete `view` types (the remainder)

Completes §4. `view` imports only the stdlib (P1: no `meta`/`typemap`/`pipeline`).

```go
type Constant struct {
    GoName, ExternName string
    Doc  DocBlock
    Kind ConstKind            // how the symbol's value is produced
}
type ConstKind int
const (
    ConstCFRef    ConstKind = iota // const CF<T>Ref → obj.Wrap(purego.CFConstant(_symbol(name)))
    ConstNSString                  // NSString* global → *String (foundation) or obj.Object (else)
    ConstObjcID                    // typedef'd NSString via symbol address → objc.ID accessor
)

type Func struct {               // a C-function wrapper (vmnet, CoreFoundation funcs, …)
    GoName, CName string
    Doc      DocBlock
    Params   []Param
    Returns  []NamedReturn        // named when >1 (E7)
    Dispatch CFuncDispatch
}
type CFuncDispatch struct {
    VarName string               // "_fnSecItemCopyMatching" (the bound func var)
    ABIIn   []string             // C ABI param types for RegisterLibFunc ("objc.ID","int", …)
    ABIOut  string               // C ABI return type ("" for void)
    Args    []string             // marshaled call args, selector order
    Outs    []OutParam           // CF/value out-refs lifted to returns
    Result  Result
    OSStatus bool                // return is OSStatus → purego.NewOSStatus(int(_rc)).Err()
    CFError  bool                // trailing CFErrorRef* out-param → errkit.FromCFError
}

type Example struct{ Name, Body string }   // curated example_test.go funcs (hand-authored; preserved)
type MockIface struct{ Name string; Lines []string } // "BootLoaderable"; method-signature lines (own+promoted)
type SetterRef struct{ Owner, Method string }
type ProviderImpl struct{ Iface, Marker string }
type DocGroup struct{ Title string; Types []TypeRef }

// Diagnostic records why a member was dropped (P2: never silent).
type Diagnostic struct{ Framework, Symbol, Reason string }
```

`headerView` (render-only input) = `struct{ Pkg string; Imports []string }`.

## Appendix B — `gather` API and algorithms

### B.1 Public surface

```go
package gather

// Framework builds the complete, immutable view for one framework. Diagnostics list every
// member intentionally dropped (unresolvable shape); err is for unexpected internal failure.
func Framework(m *meta.FrameworkMeta, reg *pipeline.Registry, docs *appledocs.Docs) (view.Framework, []Diagnostic, error)

func Support() []view.SupportPackage   // objref/obj/rt/errkit/dispatch, from support/*.txt
```

### B.2 Context (replaces all package state — P6)

`newContext(m, reg, docs)` computes, once and up front (was the 5 caches): `pkg`, `prefix`,
`classes`, `enumNames`, `typeNames`, `structEmit`, `abstract`, `subclasses`, `setterOf`. It owns
two accumulators used only during the build and then frozen into the view: `referenced
map[string]bool` (enums any member used) and `imports *view.ImportSet`. The `Context` is a local
value in `Framework`; nothing escapes it.

### B.3 Resolver functions (one per construct; pure, take `*Context`)

```go
func (c *Context) class(m meta.Class) (view.Class, error)
func (c *Context) ctors(m meta.Class, abstract bool) []view.Ctor          // none if abstract (WS3)
func (c *Context) setters(m meta.Class) []view.Setter
func (c *Context) methods(m meta.Class) []view.Method
func (c *Context) method(m meta.Method, owner string) (view.Method, bool)  // false ⇒ diagnostic
func (c *Context) dispatch(m meta.Method, owner string) (view.Dispatch, []view.NamedReturn, bool)
func (c *Context) param(objcType, name string) (view.Param, bool)
func (c *Context) result(objcType string) (view.Result, []view.OutParam, hasErr bool, ok bool)
func (c *Context) typeRef(goType string) view.TypeRef                      // also records the import
func (c *Context) base(m meta.Class) *view.TypeRef                         // embedding (WS2)
func (c *Context) providerImpls(m meta.Class) []view.ProviderImpl
func (c *Context) docBlock(sym Symbol) view.DocBlock                       // Symbol = class|method|prop|enum|...
func (c *Context) provider(base string) view.Provider
func (c *Context) enums() []view.Enum                                      // referenced ∪ *ErrorCode
func (c *Context) structs() []view.Struct
func (c *Context) constants() []view.Constant
func (c *Context) funcs() []view.Func
func (c *Context) sentinels() []view.ErrorSentinel
func (c *Context) packageDoc() view.PackageDoc
```

### B.4 Naming rules (one source of truth — moved from scattered `naming.*` calls)

| Symbol | Rule | Example |
|---|---|---|
| package | `naming.PackageName(fw)` (lower, no `_`) | `virtualization` |
| class/type | de-prefix(ObjC name) | `VZMacOSBootLoader`→`MacOSBootLoader` |
| enum type & members | de-prefix | `VZVirtualMachineStateRunning`→`VirtualMachineStateRunning` |
| method | `naming.MethodName(selector)`; `…UsingBlock`→`…Using`; trailing `…error:`+`bool`→drop `Error` | `objectAtIndex:`→`ObjectAtIndex` |
| constructor | `New`+type+`With`+selector tail | `NewMacOSBootLoaderWithURL` |
| setter | `With`+prop (fluent); `Set`+prop (unless suppressed) | `WithBootLoader` / `SetBootLoader` |
| provider iface | base+`Provider` | `BootLoaderProvider` |
| marker method | `is`+base (unexported) | `isBootLoader` |
| mock iface | type+`able` | `MacOSBootLoaderable` |
| adopt fn | lowerFirst(type)+`Adopt` (unexported) | `macOSBootLoaderAdopt` |
| sentinel | `Err`+(member with domain-stem stripped) | `ErrInvalidDiskImage` |
| param | `safeParamName(naming.ParamName(name))` (keyword/alias-shadow safe) | `obj`→`obj_` |

### B.5 De-prefix + collision (deterministic — P6, no order dependence)

`prefix = detectClassPrefix(fw)` (the common leading run of the class names, e.g. `VZ`).
`deprefix(name)` strips it unless the result is empty / lower-case / a digit. **Collision rule:**
all enum type names are reserved up front (so a class method can never claim an enum's name);
if two enums de-prefix to the same Go name, the second keeps the prefix and a `Diagnostic` is
recorded. No mutation outside the `Context`.

### B.6 Abstract detection & hierarchy (WS2)

- `isAbstract(C)` ⇔ some same-framework class has `Super == C` (no metadata flag).
- `base(C)`: if `C.Super` is a same-framework, emitted class ⇒ `&TypeRef{GoName: deprefix(Super)}`
  (embed it); else ⇒ `nil` (`EmbedsRoot=true`, embed `objref.Handle`). The chain is the acyclic
  superclass DAG within one package ⇒ no import cycles.
- `subclasses(B)` = `[deprefix(K) for K in classes if K.Super==B]`, sorted.
- `setterOf` is built by scanning every class's settable polymorphic properties whose type is an
  abstract base `B`, recording `SetterRef{Owner, "With"+B}` — used for the "pass it to …" doc.

### B.7 Dispatch decomposition (the brain) — per style

`dispatch(m, owner)` chooses the style and returns a `view.Dispatch` + the named-return clause:

1. **Async** if `isAsyncCompletion(m)` (trailing completion-handler block, no non-error result
   or exactly one result param): `Style=Async`, `Await="rt.Await"` (error-only) or
   `"rt.AwaitValue"` (one result), `BlockArgs`/`BlockBody` marshal the block params (error→
   `errkit.FromObjC(purego.NSErrorToError(_pN))`, result→wrap), `send` builds the
   `objc.Send[objc.ID](recv, sel, args…, _block)`. Returns `(ctx) error` or `(ctx) (R, error)`.
2. **Slice** if the return `looksLikeNSArray` and there are no params (getter): `Style=Slice`,
   `ElemType`+`ElemConv` from `arrayElemConv`, `send` returns `objc.ID`, body wraps with
   `purego.NSArrayToSlice`. Returns `[]E`.
3. **bool+NSError** if return is `BOOL` and `IsNSError`: collapse to `error` (drop the bool);
   `Result.Kind=RVoid`, `Error=true`. Returns `error`.
4. **Plain** otherwise: for each param, `param()`; a `*scalar`/`*enum` param becomes an
   `OutParam` (lifted, `unsafe.Pointer(&_outN)` arg); `NSError**` sets `Error`; `result()` fills
   `Result` (kind/wrap/zero); add `Guards` (e.g. bounds-checked index accessors). The named
   return clause = `[Result?] + Outs + [error?]`, **named iff >1** (E7).

All branching lives here; the templates render the resulting data. The current
`plainMethodBody`/`plainMethodBodyWithOuts`/async/slice string builders are deleted.

### B.8 Type resolution (`param`/`result`/`typeRef`)

Implements the §5.2 table as pure functions. `typeRef(goType)`:
1. builtin / same-package ⇒ `Import=nil`, `DocLink="[GoName]"`.
2. cross-package (owned by another framework) ⇒ `Import=&{Alias:pkg, Path:idiomaticPrefix+pkg}`,
   `DocLink="[pkg.GoName]"`, and `c.imports.Add(Import)`.
3. records nothing twice (the `ImportSet` dedups).
`param` returns `view.Param{GoName, Type, Arg, IsOut}` where `Arg` is the marshal expression
(`purego.NSString(s)`, `objref.IDOf(x)`, `rt.FileURL(s)`, `purego.SliceToNSArray(…)`, or the bare
name for scalars/enums/value-structs). `result` returns the `Result` (kind + `Wrap` template +
`Zero`) and any lifted `OutParam`s, plus whether an `NSError**` was present.

### B.9 Import accumulation (replaces `usedImports` text-scan)

`ImportSet` is an insertion-deduped set; `List()` returns lines sorted by path then alias, with
the alias omitted when it equals the last path segment (matches today's `importLines`). A class
file's imports = the class's accumulated set; a grouped file's = the union over its members. No
rendered text is ever scanned (P6/P7).

### B.10 Docs & links (`docBlock`) (WS4)

`docBlock(sym)` = `cleanDoc(Apple Summary)` + `cleanDoc(Apple Discussion)` (strip `@abstract`/
`@discussion`/`@see`/… ) + generated `Usage` + resolved `Links`. `Usage` is composed from the
hierarchy facts: abstract base → "Abstract base — don't construct it. Concrete types:
<subclass links>. Pass one to <consumer setter link>."; concrete → "Construct with [NewX]; pass
to <setter link>. Embeds [Base]." `Links` carries the `[Type]`/`[pkg.Type]` references the
`comment` FuncMap renders; cross-package links also `imports.Add` their package.

## Appendix C — `render` driver, body composition, remaining templates

### C.1 Driver

```go
package render
func Framework(v view.Framework, dir string) error
```

Algorithm (pure function of `v`+`dir`; the only state is the read-only embedded template set):

1. `doc.go` ← exec `docfile`(`v.Package`).
2. `<pkg>_runtime_generated.go` ← exec `runtime`(`v.Package`).
3. for each `class` in `v.Classes`: `body` ← exec `class`; `file` ← exec `header`(headerView{pkg,
   class.Imports.List()}) + body; `emit.WriteGoFile(<Class>_generated.go, file)` (gofmt — E1).
4. grouped files (`providers`,`enums`,`structs`,`constants`,`funcs`,`errors`): `imports` = union of
   the group members' `Import`s; `body` ← exec the construct template over the slice; write.
5. `verifyHermetic(dir)` (the existing gate) — defence in depth; gather already guarantees it.

Imports are **never scanned**: each `view.Class.Imports` was accumulated in gather as types were
resolved, including the fixed runtime imports the templates use, added only when actually needed
(`errkit`+`unsafe`+`purego` iff any method has `Error`; `context` iff any async; `obj` iff any
`obj.Object`; etc.).

### C.2 Unified body composition (`send`, `dispatch_tail`, `BodyView`)

Plain methods, class functions, error-returning constructors, and C-functions share one tail
template. The differ is only the *call expression*; gather/FuncMap computes it, then a tiny
wrapper carries both:

```go
type BodyView struct { D view.Dispatch; Call string }   // render-only
```

`send(d)` (FuncMap) for ObjC dispatch, `cfuncCall(d)` for C functions:

```text
send(d):      args := [d.RecvExpr, fmt("objc.RegisterName(%q)", d.Selector)] + d.Args
              if d.Error { args += "unsafe.Pointer(&_nsErr)" }
              return fmt("objc.Send[%s](%s)", d.SendType, join(args, ", "))
cfuncCall(d): args := d.Args; if d.Error { args += "unsafe.Pointer(&_cfErr)"/"&_nsErr" }
              return fmt("%s(%s)", d.VarName, join(args, ", "))
```

The shared tail (early-return error path, then success; P3/E8):

```gotemplate
{{define "dispatch_tail" -}}
{{range .D.Guards}}{{.Expr}}
{{end}}{{range .D.Outs}}var {{.Local}} {{.GoType}}
{{end}}{{if .D.Error}}var _nsErr uintptr
{{end}}{{if hasResult .D}}_r := {{.Call}}
{{else}}{{.Call}}
{{end}}{{if .D.Error}}if _nsErr != 0 {
	return {{retZeros .D}}errkit.FromObjC(purego.NSErrorToError(objc.ID(_nsErr)))
}
{{end}}return {{retValues .D}}
{{- end}}
```

`plain` `method_body` (§6.3) becomes just `{{template "dispatch_tail" (bodyView .Dispatch (send .Dispatch))}}`;
`func`/`classfunc` use `cfuncCall`. `retValues(d)` = the success list `[wrap(Result,"_r")?] +
[out.Local…] + [", nil" if Error]`; `retZeros(d)` = the error-path zero prefix `[Result.Zero?] +
[out.Zero…]` ending in `, ` (empty when error-only).

### C.3 Remaining templates (full text)

```gotemplate
{{define "func" -}}
var {{.Dispatch.VarName}} func({{join .Dispatch.ABIIn ", "}}) {{.Dispatch.ABIOut}}

{{comment .GoName .Doc}}
func {{.GoName}}({{params .Params}}){{returns .Returns}} {
	_loadOnce.Do(_loadLibrary)
	if {{.Dispatch.VarName}} == nil {
		ebipurego.RegisterLibFunc(&{{.Dispatch.VarName}}, _lib, {{printf "%q" .CName}})
	}
{{template "dispatch_tail" (bodyView .Dispatch (cfuncCall .Dispatch))}}
}
{{end}}

{{define "classfunc" -}}
{{comment .GoName .Doc}}
func {{.GoName}}({{params .Params}}){{returns .Returns}} {
{{template "dispatch_tail" (bodyView .Dispatch (send .Dispatch))}}
}
{{end}}

{{define "mock" -}}
// {{.Name}} is the interface implemented by the wrapper, for mocking and dependency injection.
type {{.Name}} interface {
	obj.Object
{{- range .Lines}}
	{{.}}
{{- end}}
}
{{end}}

{{define "constant" -}}
{{comment .GoName .Doc}}
{{- if eq .Kind 0 /*ConstCFRef*/}}
func {{.GoName}}() obj.Object { return obj.Wrap(purego.CFConstant(_symbol({{printf "%q" .ExternName}}))) }
{{- else if eq .Kind 1 /*ConstNSString*/}}
func {{.GoName}}() *String { return StringFromID(purego.CFConstant(_symbol({{printf "%q" .ExternName}}))) }
{{- else}}
func {{.GoName}}() objc.ID { return purego.CFConstant(_symbol({{printf "%q" .ExternName}})) }
{{- end}}
{{end}}

{{define "docfile" -}}
// Code generated by go-bindings-codegen. DO NOT EDIT.

//go:build darwin

// Package {{.Name}} {{.Doc.Summary}}
//
// {{.Doc.Workflow}}
//
// # Types
{{range .Doc.Groups}}//
// {{.Title}}: {{typeLinks .Types}}
{{end -}}
package {{.Name}}
{{end}}
```

`runtime` is today's dlopen bootstrap template, unchanged.

### C.4 FuncMap additions (body composition) — extends §6.5

| Func | Signature | Returns |
|---|---|---|
| `send` | `send(view.Dispatch) string` | ObjC `objc.Send[…](…)` call (C.2) |
| `cfuncCall` | `cfuncCall(view.Dispatch) string` | bound C-func call (C.2) |
| `bodyView` | `bodyView(view.Dispatch, call string) BodyView` | the tail wrapper |
| `hasResult` | `hasResult(view.Dispatch) bool` | `Result.Kind != RVoid` |
| `retValues` | `retValues(view.Dispatch) string` | success return list (C.2) |
| `retZeros` | `retZeros(view.Dispatch) string` | error-path zero prefix (C.2) |
| `typeLinks` | `typeLinks([]view.TypeRef) string` | `[A] · [B] · …` for the doc index |
| `join` | `join([]string, sep string) string` | strings.Join |

Still pure (P7); the FuncMap-set test (§6.5) is updated to this exact union.

## Appendix D — Tests & gates (concrete)

### D.1 Golden harness

`gather/golden_test.go`: for each fixture dir, `Framework(load(input)) ` → compare the returned
`view.Framework` against `want.json` with `go-cmp` (struct equality); `-update` rewrites
`want.json`. `render/golden_test.go`: `render.Framework(want.Framework, tmp)` → compare each
emitted file byte-for-byte against `testdata/<file>.golden`; `-update` rewrites. Fixtures per §6.6.

### D.2 Architecture & conformance gates (each a `Test*` that fails CI)

```text
TestNoStringBuildingOfGo   — AST-scan gather/ + render/: no fmt.Fprintf/Sprintf/bytes.Buffer/
                             strings.Builder whose result is a Go declaration (allow-list: send/
                             cfuncCall/retValues live in render and emit *expressions*, asserted
                             to be the only ones).                                   (P7)
TestNoPackageState         — AST-scan emit/idiomatic/**: no package-level var of mutable map/slice
                             type; specifically none keyed by *meta.FrameworkMeta.   (P6)
TestRenderImportsClean     — render/ imports neither meta nor typemap.               (P1)
TestFuncMapExact           — render FuncMap keys == the §6.5+C.4 union (no logic creep). (P7)
TestGofmtClean             — gofmt -l over the regenerated tree is empty.             (E1)
TestDocCommentsStartName   — every exported decl in the tree has a doc whose first word is its name. (E2)
TestNoUnderscoreExports    — no exported identifier contains '_'.                     (E6)
TestNamedReturns           — every method with >1 return value or an out-param uses named returns. (E7)
TestInterfaceAssertions    — every concrete wrapper has var _ <able> and var _ <Provider> = (*T)(nil). (E11)
TestErrorStringStyle       — gather/render error literals are lowercase, no trailing punctuation. (E12)
TestVetLint                — go vet + golangci-lint clean on the test-bed packages.
```

### D.3 Behaviour-equivalence gate (steps 1–4)

A `make regen-diff` target regenerates the whole tree to a temp dir and `git diff --no-index`
against the committed tree; **must be empty** through step 4 (structure-only refactor). Steps
5a–5d each update goldens deliberately and re-assert the global build + hermetic + example gates.

## Appendix E — Edge-case rules (the exact gather decision for each)

1. **Generic collections** (`NSArray<E>`/`NSSet`/`NSDictionary` *as a wrapper type*) stay uniform
   `struct{ objref.Handle }`; a generic-element *return on that wrapper* is `obj.Object`. As a
   *parameter/return of another method*, `NSArray<E>` resolves to `[]E'` (Slice dispatch).
2. **Cross-framework base** ⇒ `base(C)=nil`, `EmbedsRoot=true` (flat root; no cross-package
   embedding ⇒ no import cycle).
3. **Cross-framework value struct** ⇒ `typeRef` owner-qualifies (`corefoundation.CGRect`) and
   `imports.Add`, **only if** `reg.EmittableStructs[name]`; otherwise the member is dropped with a
   diagnostic (never reference a struct the owner won't emit).
4. **CF "…Ref"**: resolve the Go type — pointer + `isCFObjectType` ⇒ `obj.Object`; integer handle
   (e.g. `MIDIObjectRef`→`uint32`) ⇒ scalar.
5. **OSStatus C-function** ⇒ `CFuncDispatch.OSStatus=true` ⇒ returns `error` via
   `purego.NewOSStatus(int(_rc)).Err()`.
6. **`CFErrorRef *` / `NSError **` out-param on a C-function** ⇒ `CFError`/error path ⇒
   `errkit.FromCFError`/`errkit.FromObjC`.
7. **Value/scalar out-params** ⇒ lifted to `OutParam` (named returns, E7).
8. **Async** ⇒ `AwaitValue` (one typed result) or `Await` (error-only).
9. **`BOOL`+`NSError`** ⇒ collapse to `error`; drop the bool; strip a trailing `Error` from the
   name when the selector ends `error:`.
10. **Block param/return** ⇒ Go `func(...)` + `objc.NewBlock` adapter (`idiomaticBlockParam`);
    block shapes not expressible ⇒ drop the method with a diagnostic.
11. **Abstract base** ⇒ derived (some same-framework class has `Super==C`); no metadata flag.
12. **De-prefix collision** ⇒ enum names reserved first; on clash keep the prefixed name +
    diagnostic.
13. **Param name shadowing** (`obj`/`rt`/`errkit`/`objref`/`purego`/`objc`/`context`/keywords) ⇒
    `safeParamName` suffixes `_`.
14. **Reserved member names** ⇒ a generated method/setter named `Description`/`IsEqual`/`IsKind`/
    `Handle`/`String` is dropped (provided by the embed/obj.Object, E13); recorded.
15. **Hand-authored files** ⇒ `scanHandAuthored` collects `(receiver, name)`; gather omits any
    member a human already wrote; render never overwrites a non-`_generated.go` file;
    `example_test.go` is preserved.
16. **NSString constant in a non-foundation package** ⇒ `ConstObjcID` (objc.ID accessor), not
    `*String` (the foundation-local wrapper isn't importable cross-package).
17. **Variadic** ⇒ only format-variadics emitted (matches the raw layer); others dropped.
18. **Multiple inits / designated initializers** ⇒ one `Ctor` each; plain `+new` only for
    non-abstract classes.
19. **Selector→same Go name within a class** ⇒ dedupe, keep first, diagnostic.
20. **Provider with a cross-framework subclass** ⇒ providers are same-framework only (the marker
    is unexported); a cross-framework subclass crosses the boundary as `obj.Object`.

## Appendix F — Pipeline & support wiring

`GenerateIdiomatic` (`pipeline/generator.go`):

```go
reg  := cfg.Registry                       // built once, with EmittableStructs
docs := cfg.AppleDocs                      // merged once
render.Support(gather.Support(), supportDir)        // objref/obj/rt/errkit/dispatch
var allDiags []gather.Diagnostic
for _, fw := range reg.Frameworks {
    if skip(fw) { continue }               // libraries / swift-only / filtered
    v, diags, err := gather.Framework(fw, reg, docs)
    if err != nil { return err }
    if err := render.Framework(v, outDir(fw)); err != nil { return err }
    allDiags = append(allDiags, diags...)
}
writeDiagnostics(allDiags)                  // metadata/idiomatic-diagnostics.json (P2: dropped is visible)
```

- **Diagnostics ratchet**: like the existing `diagnostics-baseline.json`, CI fails when a *new*
  member is dropped beyond the committed baseline — so "dropped with a diagnostic" can't silently
  grow (P2/P11).
- **Support packages** (`objref`/`obj`/`rt`/`errkit`/`dispatch`) keep being emitted from
  `support/*.txt`, now via `render.Support`; the recent `obj.As`/`obj.Bytes`/`obj.ID` additions
  stay.
- **Legacy retirement**: during steps 1–4, `EmitFrameworkWrappers` delegates to
  `render.Framework(gather.Framework(...))`; the 5 caches, `referencedEnums`, `usedImports`, and
  every string builder are deleted as their construct is ported; the old `idiomatic` package is
  removed once empty.

---

## Definition of done — "bottomed out"

The refactor is complete when **all** of the following hold (each has a gate in §8/App. D):

1. Every emitted construct (class, ctor, setter, method (plain/async/slice/bool-NSError/
   out-param), provider, enum, struct, constant, C-func, sentinel, doc.go, runtime) is produced
   by **gather (decide) → template (render)**; **no `fmt.Fprintf` of Go declarations** remains.
2. **No package-level mutable state** in the emitter; all per-framework data lives in
   `gather.Context`.
3. **Imports are computed** from resolved `TypeRef`s; `usedImports` text-scanning is deleted.
4. The whole tree regenerates **byte-identical** through step 4 (structure-only), then the
   readability features (WS2 embedding+sealed providers, WS3 prune, WS4 docs/links/overview,
   WS5 examples) land in 5a–5d, each behind the global build + hermetic + golden gates.
5. Every Zen (P1–P11) and Effective Go (E1–E14) rule maps to a passing gate (§2e); the principle
   tensions (§2d) are resolved as recorded.
6. `go doc` of `virtualization` and `virtualization.BootLoader` reads like a manual; the example
   packages compile/run; the sealed-provider negative compile test passes.
7. The full 244-framework regen builds with 0 failing packages, 0 hermetic violations,
   `bindings/frameworks` unchanged, and the diagnostics ratchet is green.
