# Phase 4 — Opaque handle typedefs → named Go handle types

Kickoff/implementation plan for the `feat/opaque-handle-types` branch. Grounded in a
full read of the codebase (line numbers as of `main` after PR #95). Written to be
executed in focused slices, each with its own regen + parity/determinism gate.

Prior context: this follows the Phase-3 enum-width work and the `UInt32`-typedef
struct-field width fix (both merged). The Phases roadmap lives in
`IDIOMATIC_STRUCT_LAYOUT_PROGRESS.md`.

---

## Goal

Surface opaque handle typedefs (`typedef struct Foo *FooRef`) as **named Go handle
types** with a `.Ptr()` accessor, instead of degrading them to `unsafe.Pointer`, for
both parameters and returns — so callers get type safety instead of a bare pointer.

## Confirmed lifetime model (decided with the user)

**"Full lifetime where it's real, named-types where it isn't."**

- The gap handles are **mostly NOT reference-counted** (`AudioQueueRef` is created by
  `AudioQueueNewOutput`, destroyed by `AudioQueueDispose` — no retain/release). For
  these, emit a **named type with manual lifetime** (no finalizer): behavior-identical
  to today's `unsafe.Pointer`, just typed. Attaching an auto-release finalizer to one
  of these would **double-dispose → crash**.
- **Full retain/adopt/finalizer lifetime** applies **only** to genuinely
  reference-counted handles (CF-registered, objc). Most of those are *already* handled
  (CF refs → `obj.Object` with `obj.Wrap`/`obj.Adopt`; libraries CF → `NewX`+`cgo.Track`).
  The remaining lifetime work is to **unify the two inconsistent already-retained
  signals** onto `ReturnType.IsAlreadyRetained`.

## The gap (quantified)

- **Frameworks (the real target): ~221** non-CF opaque `…Ref` handles degrade to
  `unsafe.Pointer` (of 244 `…Ref` opaque-pointer typedefs; the rest are CF-registered).
  Examples: `AudioQueueRef`, `AudioConverterRef`, `AudioQueueTimelineRef`,
  `AUEventListenerRef`, `CAClockRef`, `CMTagCollectionRef`, `SCDynamicStoreRef`,
  `SCNetworkReachabilityRef`, AE stream refs. Diagnostics baseline
  (`metadata/diagnostics-baseline.json`): 1726 "unresolved pointer type" + 1171
  "unresolved named type" → unsafe.Pointer; ~221 are the true non-CF handle subset,
  +281 `…Ref *` double-pointer out-param forms. `unsafe.Pointer` appears 10,836× in
  framework signatures (not all handles — many are `void*` context/refcon).
- **Libraries: negligible** (19 non-`_t` opaque pointer typedefs, mostly junk like
  `Class`/`SEL`/`id`; only ~1 real, e.g. `IOReportSubscriptionRef`). Do NOT invest a
  slice here; fold `IOReportSubscriptionRef` into slice 1's libraries `handleBaseName`
  extension only if trivial.

The backing zero-field opaque structs (`type OpaqueAudioQueue struct{}`) are already
emitted but **never surfaced as typed pointers** (`*Opaque…` count in raw framework
signatures = 0) — their refs degrade.

---

## Current mechanisms (with file:line, the code to build on/generalize)

### Frameworks (purego) mapper — where handles resolve
`internal/codegen/frameworks/typemap/mapper.go`
- `resolveNamed` (640) for a bare `FooRef`; `resolvePointer` (315) for the `struct Foo *`
  body / `FooRef *` out-param. Outcomes: (a) CF wrapper via `IsCoreFoundationOpaqueRef`
  (879-889) + `CFTypeIndex` (72-73, fallback 718-725); (b) `objc.ID` (blocked/unavailable);
  (c) `*SomeStruct` when the pointee is in `StructIndex` (375-381); **(d) `unsafe.Pointer`**
  — the degraded gap (399-400, 727-728).
- `CFTypeIndex` populated in `internal/codegen/frameworks/pipeline/loader.go:335-365`
  (any non-CF typedef whose target is `struct X *` with `X` a zero-field struct). **This
  is where an "opaque handle registry" is already half-built** — the gap handles are
  exactly the zero-field-struct pointer typedefs NOT already re-typed.

### The template to generalize: `osObjectRegistry`
`internal/codegen/emit/idiomatic/frameworks/typeresolve.go`
- Hardcoded 11-entry map (122-134) → concrete library types. `osObjectLibraryType`
  (140-154). Param emission in `idiomaticArg` (269-278): signature type `dispatch.Queue`,
  call arg `objc.ID(uintptr(param.Ptr()))`. Return in `idiomaticRet` (570-573): adopt via
  `obj.Adopt` when `isOSObjectReturn` (883-890, sniffs `OS_OBJECT_RETURNS_RETAINED`).
- CF param/return re-typing: `isCFObjectType` (861-881) → `obj.Object`; param
  `objref.IDOf(param)` (292-296); return `obj.Wrap(%s)` (563-566).

### The handle-wrapper template (shape to emit)
Libraries `handle_wrapper` — `internal/codegen/emit/idiomatic/libraries/render/templates/cfunctions.tmpl:16-45`:
```gotmpl
type {{.GoName}} struct{ ptr unsafe.Pointer }
func Wrap{{.GoName}}(p unsafe.Pointer) {{.GoName}} { return {{.GoName}}{ptr: p} }
func (h {{.GoName}}) Ptr() unsafe.Pointer { return h.ptr }
```
Detection `handleBaseName` (`emit/idiomatic/libraries/cfunctions.go:245-269`) — currently
requires `_t` suffix + resolves-to-`unsafe.Pointer` + not enum/struct/block. CF wrapper
(pointer + `NewX`+`cgo.Track`) — `emit/raw/libraries/render/templates/structs.tmpl:21-48`.

### Lifetime primitives (for the refcounted slice)
- Frameworks: `bindings/runtime/obj/obj_generated.go` — `Wrap(id)` (retain +1 then Track),
  `Adopt(id)` (take existing +1, NO retain, then Track), `Object.Release()`. `Track` =
  `runtime.SetFinalizer` sending `-release` (`bindings/runtime/purego/objc.go:50-72`).
- Libraries: `bindings/runtime/cgo/memory.go` — `Retain`/`Release`, `Track` (SetFinalizer,
  **dispatches release to the main queue** for AppKit dealloc safety, 36-51), `KeepAlive`.
- `IsAlreadyRetained` set in `internal/scanner/extract.go:makeReturnType` (620-640:
  new/alloc/copy/mutableCopy + NS/CFReturnsRetained). Consumed today ONLY in the raw
  bridge (`emit/raw/libraries/bridge.go:517-526`) and raw frameworks classes
  (`emit/raw/frameworks/classes.go:536`). The idiomatic os_object path uses the parallel
  `OS_OBJECT_RETURNS_RETAINED` string-sniff — **unify these**.
- `KeepAlive` discipline is pervasive (setters/constructors/methods emit
  `defer runtime.KeepAlive(...)`); handle params passed as raw pointers MUST join it.

---

## Slices (implement one at a time; regen + `go build ./bindings/...` + parity 0 +
## determinism after each)

### Slice 1 — Frameworks opaque-handle registry + type emission (type-safety, no lifetime)
The bulk of the value. Non-refcounted handles get a named type with manual lifetime.

1. **Registry.** Add a computed set of opaque-handle typedefs to the frameworks mapper
   (mirror/extend `CFTypeIndex`): a typedef `FooRef` whose target is `struct Foo *` where
   `Foo` is a **zero-field** struct, that is NOT already CF (`IsCoreFoundationOpaqueRef` /
   `CFTypeIndex`) and NOT objc. Populate in `loader.go` alongside `CFTypeIndex` (335-365).
   Name: `OpaqueHandleIndex map[string]string` (typedef → owning framework).
2. **Emit the handle type** in the idiomatic framework package (hermetic — emit locally,
   never import raw). New emitter, e.g. `emit/idiomatic/frameworks/handles.go`, writing
   `type FooRef struct{ ptr unsafe.Pointer }` + `func (h FooRef) Ptr() unsafe.Pointer` +
   `func WrapFooRef(p unsafe.Pointer) FooRef` for every handle the framework uses (param
   or return). Dedup against `takenNames`; skip collisions with an emitted class/enum/struct.
   Wire into the idiomatic per-framework emit sequence next to enums/structs.
3. **Param wiring** — `idiomaticArg` (typeresolve.go, near the osObject branch 269-278):
   an `OpaqueHandleIndex` param → signature type `FooRef`, call arg the raw pointer
   (`unsafe.Pointer(param.Ptr())` — NOT `objc.ID`, these are C handles not objc objects;
   check what the raw signature expects, which is `unsafe.Pointer`). Add
   `defer runtime.KeepAlive(param)`.
4. **Return wiring** — `idiomaticRet` (near 560-573): an `OpaqueHandleIndex` return →
   `WrapFooRef(%s)`. No adopt/retain here (non-refcounted).
5. **Cross-framework handles**: a handle owned by framework A used in framework B must be
   referenced as `a.FooRef` (import A's idiomatic package) — reuse the
   `crossFrameworkEmittedStruct`/prefix machinery; keep hermetic (idiomatic→idiomatic).
6. Generalize/retire the hardcoded `osObjectRegistry` where the computed registry covers it
   (but os_object handles map to *library* types, a different case — keep that bridge, or
   fold it in carefully).

Verify: build all bindings, parity 0, determinism, and spot-check that e.g.
`AudioQueueDispose(inAQ AudioQueueRef)` and `AudioQueueNewOutput(... outAQ *AudioQueueRef)`
generate (the out-param `Ref *` form is slice 2).

### Slice 2 — Out-param (`FooRef *`) double-pointer forms
The ~281 `…Ref *` out-params (e.g. `AudioQueueNewOutput(..., AudioQueueRef *outAQ)`).
Surface as `*FooRef` (pointer to handle) or an out-return; pass `&_tmp` and `*outAQ =
WrapFooRef(_tmp)` after the call. Careful with the purego ABI (out-params are pointers to
pointers).

### Slice 3 — Lifetime unification (refcounted handles only)
1. Replace the `OS_OBJECT_RETURNS_RETAINED` string-sniff (typeresolve.go:883-890) with
   `ReturnType.IsAlreadyRetained` end-to-end, so returns choose `obj.Adopt` (+1 already
   held) vs `obj.Wrap` (borrow → retain) consistently for CF/objc/os_object returns.
2. Give the libraries `handle_wrapper` a real lifetime story where the handle IS
   reference-counted: it is a value type today (`WrapX`, no finalizer, cannot `SetFinalizer`).
   Either convert refcounted library handles to pointer handles (`*FooRef` + `NewX`/`AdoptX`
   + `cgo.Track`) or add explicit `Retain`/`Release`. Do NOT touch non-refcounted handles.
3. Keep the `KeepAlive` discipline for every handle param passed as a raw pointer.

### Slice 4 (optional) — Libraries `…Ref` + `IOReportSubscriptionRef`
Extend libraries `handleBaseName` (cfunctions.go:245-269) beyond `_t` to opaque `…Ref`
typedefs resolving to `unsafe.Pointer`; reuse `handle_wrapper`/`wrapper_func`. Low value
(19 typedefs, ~1 real) — only if cheap.

---

## Memory-safety priorities (do not skip)
1. **Never** attach an auto-release finalizer to a non-refcounted handle (double-dispose).
   Slice 1 handles get NO finalizer.
2. Preserve the `obj.Wrap` (retain) vs `obj.Adopt` (adopt) distinction; drive it from
   `IsAlreadyRetained` (slice 3), never guess.
3. Frameworks handle types stay **hermetic** — emitted in the idiomatic package, never a
   raw import (the idiomatic frameworks layer must not reference `bindings/internal/raw`).
4. Maintain `runtime.KeepAlive`/`cgo.KeepAlive` for handle args passed as raw pointers, so
   the collector cannot finalize a wrapper mid-call.
5. Any library finalizer must go through `cgo.Track` (main-queue release for AppKit safety).

## Verify (per slice)
```sh
go build $(go list ./... | grep -v acceptance)   # or: go build ./bindings/...
go run ./cmd/generate/ bindings && go run ./cmd/generate/ idiomatic
go run ./cmd/generate/ parity          # must stay 0 missing
go test ./internal/codegen/... ./internal/scanner/...
# determinism: regen twice → identical
```
No re-scan is needed for slices 1-3 (pure codegen; the metadata already has the typedefs,
zero-field structs, and `IsAlreadyRetained`). Slice 4 needs no scan either.
