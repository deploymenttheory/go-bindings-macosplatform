# Idiomatic layer — typed-struct / layout work: progress & next-wave plan

Handoff doc for the ongoing effort to grow the idiomatic layer's coverage, type more
C structs, cut `unsafe.Pointer`, and (eventually) retire the raw layer. Written after
Phase 2 STEP A. Everything below is grounded in the actual codebase as of `main`
@ `e56e3f3f1`.

---

## Overarching goal & context

The idiomatic layer (`bindings/frameworks/*`, `bindings/libraries/*`) is the only public
API; the raw layer lives under `bindings/internal/raw` and is meant to be retired once
idiomatic coverage is mature. This effort progressively types more C structs and handle
params so callers never touch `unsafe.Pointer` or `obj.Object` where a concrete type is
possible. The consuming project is `guestweave` (a separate repo) — the original trigger
was cleaning up its `vmnet.go` / `iousbhost.go` / `coreaudiosink.go` against the fixed SDK.

Regenerate commands (fast path, no Xcode needed once metadata exists):
```
go run ./cmd/generate/ scan --framework all      # ~3.5 min; ~7 min with the STEP A clang layout pass
go run ./cmd/generate/ bindings                  # raw layer (bindings/internal/raw)  ← easy to forget!
go run ./cmd/generate/ idiomatic                 # idiomatic layer (bindings/{frameworks,libraries})
go run ./cmd/generate/ parity                    # must stay "0 missing"
```
Verify: `go build $(go list ./... | grep -v acceptance)` is NOT how this repo checks — use
`go build ./bindings/...` (whole tree compiles = the key regression gate) + `go run ./cmd/generate/ parity` +
`go test ./internal/codegen/... ./internal/scanner/...` + determinism (regen twice → identical).

---

## Completed & MERGED to `main`

| PR | What |
|----|------|
| #88 | in/out count params (vmnet_read/write `pktcnt`), block typedefs → Go closures (enum + `void*` args), `unsafe.Pointer` returns, zero-literal fix. |
| #90 | typed pointer-to-struct **returns** (`*Struct`) + scanner capture of referenced **external** C structs (e.g. `IOUSBDeviceDescriptor` from IOKit headers). |
| #91 | dispatch/xpc handle **params** → concrete library types (`dispatch.Queue`, `xpc.Object`) instead of `obj.Object`. |
| #92 | fixed-size **array** struct fields → Go arrays (`uint8_t[16]`→`[16]uint8`); struct-pointer **params** → `*Struct`; fixed two latent bugs (const-input wrongly lifted to out-param; method-vs-embedded-base-field name collision). |

Also earlier in the SDK: `inout_count_params` sidecar in `idiomatic.json`; the block-typedef
detection via `mapper.ResolveBlockSignature`; the `osObjectRegistry` (dispatch/xpc) in
`typeresolve.go`.

The guestweave-side workarounds these enable removing (once a release is cut): `ebipurego`
import, `objc.NewBlock` handlers, the `obj.Adopt(objc.ID(uintptr(x.Ptr())))` dance, the
`vmnet_read`/`write` purego re-bind.

---

## Phase 2 STEP A — DONE, pushed, NOT yet merged

Branch: **`feat/idiomatic-authoritative-struct-layouts`** @ `1bf24f724` (pushed, PR-ready).

Two things:
1. **Width-correction** — the type mapper resolves C `int`/`unsigned int` to Go `int`/`uint`
   (8 bytes), but a C int is 4. For a struct **field** this mislays every field after the
   first int. `structFieldGoType(objcType, mapped)` in
   `internal/codegen/emit/idiomatic/frameworks/structs.go` corrects `int`→`int32` /
   `unsigned`→`uint32` on struct fields only (params unaffected), element-wise for arrays.
   Applied in `resolveStructFields` and the `emitStructs` field loop. ~189 fields fixed.
2. **Clang authoritative-layout cross-check** — a second clang pass
   (`DumpRecordLayouts` in `internal/scanner/clang.go`, `-emit-llvm -c -Xclang
   -fdump-record-layouts-simple`, output on **stdout**, sizes/offsets in **bits**) records
   each value-used record's exact offsets + size into metadata (`Struct.Size`,
   `StructField.Offset` — added to **both** model copies: `internal/macosplatformmetadata/model.go`
   and `internal/codegen/frameworks/meta/model.go`). `ComputeEmittableStructs` then requires
   the width-corrected Go layout to reproduce the authoritative layout exactly
   (`layoutMatchesAuthoritative` + `goStructLayout`); a mismatch keeps the struct opaque.

Verified: parity 0 missing, whole tree compiles, tests pass (new `goStructLayout`,
`structFieldGoType`, `parseRecordLayouts` cases), deterministic, **0 structs excluded**
(width-correction makes all 210 clang-laid-out structs match), **no cross-pipeline cascade**
(the emittable set is unchanged, so libraries are untouched).

**Action: merge this branch to `main` before starting STEP B.**

### Key discoveries in STEP A (don't relearn these)
- **"Padding reproduces any layout" is FALSE.** Go forces natural field alignment — padding
  can only *add* space, never remove it. A packed/misaligned layout (a `uint16` at an odd
  offset) cannot be a typed Go struct + padding; it needs byte-array fields + accessors
  (a Phase-5 idea). So the clang cross-check *validates/excludes*; it does not enable
  packed-misaligned structs.
- **`-fdump-record-layouts` only covers value-used structs.** Pointer-only-referenced structs
  (the USB descriptors we care about) are NOT in the dump. So the cross-check reaches ~13%
  of structs (210/1641); the rest rely on computed layout (`layoutSafeFromGoTypes`).

---

## Phase 2 STEP B — the cross-pipeline reconciliation (NOT started; this is the next wave)

**Goal:** let **library-owned** shared C structs (`audit_token_t` used by security/bsm/
endpointsecurity, mach types, etc.) be typed as value structs and cross-referenced from
frameworks — instead of collapsing to opaque handles or dangling references. This is what
lets the scanner's `int`-admission be re-enabled safely.

**User decision on the core naming fork:** libraries adopt `ExportedTypeName` (unify: the
libraries pipeline names struct types `AuditTokenT` like the frameworks pipeline + the shared
type mapper, rather than the current `Audit_token_t` from `GoTypeName`). This is a broad
breaking change to library type-alias names.

### Why it's a real multi-layer refactor (discovered, verified)
- **The two pipelines are independent.** Libraries use a **separate typemap package**
  (`internal/codegen/libraries/typemap`, NOT `internal/codegen/frameworks/typemap`) and a
  **separate `Mapper`** with **no shared `EmittableStructs`**. (Corollary: the earlier
  cascades were via shared **metadata**, not shared mapper state.)
- **The libraries pipeline has NO value-struct emitter.** `internal/codegen/emit/idiomatic/libraries/`
  has EmitCFunctions / EmitAliases / EmitSpecs / EmitSlices / EmitAsync — none emit `type X struct`.
  Orchestrated in `internal/codegen/libraries/pipeline/idiomatic.go:emitIdiomaticLibrary`.
- **Struct-vs-handle decision** is in `cfunctions.go:handleBaseName` (~249-269): a `_t` typedef
  that the raw mapper resolves to `unsafe.Pointer` and is NOT in `framework.Structs` becomes an
  opaque handle (`AuditToken` = `struct{ ptr unsafe.Pointer }`). Once `audit_token_t` is a
  struct-with-fields in library metadata (via `int`-admission), it leaves the handle path and
  the library tries to reference it as a value struct — which today dangles.
- **Naming split:** frameworks + shared mapper use `naming.ExportedTypeName` (`AuditTokenT`,
  `internal/codegen/frameworks/naming/naming.go:170`); the libraries **raw** emitter uses
  `naming.GoTypeName` (`Audit_token_t`, keeps underscores) in
  `internal/codegen/emit/raw/libraries/structs.go:102` (+ enums/externs/functions/typedefs).
- **`StructIndex` owner** (`internal/codegen/frameworks/pipeline/loader.go:330-351`) attributes
  a multi-owner struct to the lexicographically-smallest owner **with fields** — so
  `audit_token_t` → `bsm` (a library). `qualifyType` (`mapper.go:762-793`) then uses
  `ModulePrefix` (framework prefix!), not `LibraryModulePrefix` — a latent bug.
- **Cross-ref helpers** (`typeresolve.go`): `crossFrameworkValueStruct` hardcodes
  `idiomaticFrameworkPrefix` (no library branch); `crossFrameworkEmittedStruct` *does* check
  `mapper.LibraryPkgs`. `idiomaticLibraryPrefix` constant already exists (added in #91).

### STEP B layers (implement one at a time; re-scan + `go build ./bindings/...` clean after each)
1. **Library struct-emittability + value-struct emitter.** Add an `EmitStructs` to
   `internal/codegen/emit/idiomatic/libraries/` that emits `type <ExportedTypeName> struct {…}`
   for library structs whose fields all resolve to primitives/arrays. Needs a library-side
   emittability computation (frameworks' `ComputeEmittableStructs` isn't reachable — the
   mapper types differ). Reuse the width-correction (`libStructFieldGoType`, mirror of the
   frameworks helper). Wire it into `emitIdiomaticLibrary`'s emitter table, feeding its names
   into the `declared` set so `EmitAliases` doesn't redeclare them.
   *(A first skeleton of this file was written then removed — it used the wrong `typemap`
   package. Rewrite against `internal/codegen/libraries/typemap`.)*
2. **Naming → `ExportedTypeName` in the libraries pipeline** (raw + idiomatic + aliases), so
   the library exports `AuditTokenT`. This is the broad breaking change; do it centrally where
   the libraries emitters call `GoTypeName` for TYPE names (structs/typedefs/enums). Confirm
   the raw library re-references stay consistent (declaration + uses).
3. **Library `cfunctions` value-struct params.** Teach `handleBaseName`/`buildIdiomaticParams`
   to surface a now-struct-typed param as `*AuditTokenT` and pass `unsafe.Pointer(x)` to raw.
4. **Cross-ref fixes.** `crossFrameworkValueStruct` (+ the pointer path) select
   `idiomaticLibraryPrefix` for library-owned structs; `qualifyType` uses `LibraryModulePrefix`
   for library owners; remove the temporary library guard (added then reverted in this work).
5. **Re-enable `int`-admission** in the scanner: `cScalarSizeAlign` in
   `internal/scanner/extract.go` re-adds `int`/`signed int`/`unsigned int`/`unsigned` → (4,4).
   This is what makes `audit_token_t` a captured struct in library metadata.
6. **Regenerate BOTH raw and idiomatic** (`bindings` then `idiomatic`) after the full re-scan —
   a process gap: STEP-B changes need the raw layer regenerated too, or idiomatic references to
   raw struct types dangle.

**Known trap:** re-enabling `int`-admission WITHOUT layers 1-4 in place reproduces the
compile cascade (idiomatic `endpointsecurity`/`bsm` reference an undefined
`raw.Audit_token_t` / `endpointsecurity.Audit_token_t`). Land 1-4 first, then 5, then 6.

---

## Remaining phases (from the approved roadmap, after Phase 2 concludes)

- **Phase 3** — enum fields → Go enum type; nested emittable-struct fields (recursive). Both
  frameworks-local, low cascade risk.
- **Phase 4** — opaque handle typedefs (`typedef struct X *XRef`) → named handle types for
  params *and* returns (generalize the `osObjectRegistry` idea). Lifetime (retain/adopt) care.
- **Phase 5** — unions → `[N]byte` + typed accessors; function-pointer typedef fields → Go
  func types. The byte-array+accessor technique here is also what would finally enable
  packed-**misaligned** structs (which STEP A proved padding can't do).

## Then: guestweave
Once a new SDK version is cut, rewrite guestweave's `vmnet.go` / `iousbhost.go` /
`coreaudiosink.go` to the clean idiomatic API and drop the workarounds; build + vet + test.

---

## Branch/PR state at time of writing
- `main` @ `e56e3f3f1` — #88/#90/#91/#92 merged.
- `feat/idiomatic-authoritative-struct-layouts` @ `1bf24f724` — STEP A, pushed, **merge next**.
- STEP B: not started (design above is complete and ready).
</content>
