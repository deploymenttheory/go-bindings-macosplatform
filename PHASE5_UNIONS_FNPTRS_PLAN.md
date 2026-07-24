# Phase 5 — byte-array + accessor admission tier (unions, packed-misaligned, fn-ptr fields)

Repo: `github.com/deploymenttheory/go-bindings-macosplatform`. Branch
`feat/phase5-unions-fnptrs` off `main` (@ #98). Grounded against the code
(file:line anchors verified).

## Goal

Add a **second admission tier** for value structs the current "clean typed struct"
path rejects. Today such a struct degrades field-by-field to `unsafe.Pointer`
(`structs.go` `hermeticFieldType` → `degrade()`), which loses size/typing (e.g.
`_MTLPackedFloat3` emits as `struct{ Field0 unsafe.Pointer }` — 8 bytes, should be
12). The new tier emits an **aligned `[N]byte` backing struct + typed accessor
methods** that reinterpret at each field's authoritative `StructField.Offset`, so
callers get typed field access through a pointer without the layout ever being
passed by value.

Three sub-features: (1) unions, (2) packed-misaligned structs, (3) function-pointer
typedef fields.

## Two facts that shape the whole design (verified)

1. **Parity is name-keyed, not layout-keyed** (`cmd/generate/parity.go`). Changing
   *how* a struct's fields render never affects parity; only introducing *new
   names* does. → tiers for packed/fn-ptr add **zero** parity risk. Only **unions**
   touch parity (new names), so their emission must land raw + idiomatic in lockstep.

2. **The by-value ABI gate is separate from the pointer/alias gate.**
   - `mapper.EmittableStructs` (generator.go:654) = the **by-value** gate
     (`typeresolve.go`, `functions.go` — params/returns passed by value through
     purego func vars).
   - `mapper.AllEmittedStructs` (generator.go:658) = pointer targets, field embeds,
     typedef-alias RHS.
   - **A byte-array struct is admitted to `AllEmittedStructs` but deliberately kept
     OUT of `EmittableStructs`.** They already sit outside it today.

## The make-or-break ABI constraint

A `[N]byte` **cannot be passed by value** correctly: on arm64 an all-float
aggregate (HFA/HVA, e.g. 3×`float`) goes in SSE/float registers, but purego —
seeing `[N]byte` — classifies it as integer registers (same for SSE eightbytes on
amd64). Float classification is unrecoverable from bytes. → byte-array structs
**must stay out of `EmittableStructs`** (by-value). This is both *safe* and *free*
(they're already excluded). The accessor tier upgrades their **pointer-based**
field access from `unsafe.Pointer` blobs to typed accessors, never exposing them to
a by-value register decision.

Two more ABI rules:
- **Emit `type U struct{ … [N]byte }`, never `type U [N]byte`** — purego only
  decomposes `reflect.Struct` args (arrays only as struct fields).
- **Force alignment** with a zero-size leading field: `type U struct { _ [0]uint64;
  data [N]byte }` (a `[0]uintK` has align K, size 0). Align must come from clang
  (`AlignOf`); until the scanner captures it (Commit 2), infer `min(nextPow2(Size),16)`.

## Metadata state (verified across all .gometa.json)

- `Struct.Size` populated on **298** structs; `StructField.Offset` on **556** fields
  (only where clang's field count matched — `extract.go:952`). Offset 0 is omitted
  (json omitempty) = 0.
- **Unions are dropped at scan time** — `scanStruct` early-returns on
  `TagUsed=="union"` (`extract.go:899`). No `IsUnion` flag exists. BUT clang's
  layout dump already handles union records (tag stripped, keyed by name, all
  members at offset 0) — so a *named* union yields size + all-zero offsets the
  moment `scanStruct` stops dropping it. Anonymous unions need AST recursion (later).
- No `Align` field on `meta.Struct`/`RecordLayout` (`clang.go`) — add in Commit 2.
- **packed:true** on 49 structs; ~5 are packed-**misaligned** AND sized
  (`CAFChunkHeader`, `CAFMarker`, `CAFPositionPeak`, `CAFStringID`).
- Union-typed-field structs: 51; **6 sized** (`_MTLPackedFloat3`, `_MPSPackedFloat3`,
  `AURenderEventHeader`, `cssm_db_attribute_info`, `HMHelpContent`,
  `SparseIterativeMethod`).
- Fn-ptr-field structs: 37; **0 sized** (pointer-only dispatch tables clang never
  lays out by value) → blocked on a scanner change + the `purego.RegisterFunc`
  trampoline. Last and most expensive.

## Accessor design

New pure helper in `internal/codegen/emit/structlayout` (string/int-only contract):
```go
type Accessor struct { Name string; Offset int; GoType string }
func AccessorPlan(fieldNames []string, offsets []int, goTypes []string) []Accessor
```
Generated shape (pointer receiver — zero-copy, aliases the backing store):
```go
type MTLPackedFloat3 struct { _ [0]uint32; data [12]byte } // align 4, size 12
func (u *MTLPackedFloat3) AsElements() [3]float32 {
    return *(*[3]float32)(unsafe.Pointer(&u.data[0]))
}
```
Body: `*(*T)(unsafe.Pointer(&u.data[offset]))`. Naming: `As` +
`structFieldGoName(field)` (already de-collides via initialisms). Union members
carry distinct names → no `As<Member>` collisions. Fn-ptr fields (sub-feature 3)
synthesize a Go func type and bind via `purego.RegisterFunc(&fn, codePtr)`
(`bindings/runtime/purego/dynload.go`) — purego can't call a struct-resident C
function pointer directly.

## Sequencing (independently landable, each verified)

**Commit 1 — tier scaffolding + packed-misaligned. NO scanner change, NO re-scan.**
- `structlayout.AccessorPlan` + `render.Accessors` + view type/template.
- In `ComputeEmittableStructs` (reject point ~structs.go:157): when a struct fails
  the clean layout check but has `Size != 0`, route to a new `byteArrayStructs` set
  instead of dropping. Keep it OUT of `EmittableStructs`; it's already in
  `AllEmittedStructs`.
- `emitStructs`: byte-array structs render as aligned-`[N]byte` + accessors. Align
  inferred from Size for now.
- Apply to the ~5 sized packed-misaligned structs.
- Gate: frameworks regen; `parity` = 0; build touched pkgs (audiotoolbox) — the
  metal↔compositorservices / commonpanels↔hitoolbox raw import cycles are
  PRE-EXISTING, unrelated, excluded from the check; add a `go test` reflect check
  `unsafe.Sizeof == Struct.Size` and accessor offsets.

**Commit 2 — scanner: capture unions + authoritative alignment. RE-SCAN (~6 min).**
- `scanStruct` (extract.go:899): accept `TagUsed=="union"` (drop only anonymous
  `Name==""`). Add `Align` to `RecordLayout` (clang.go) + `Struct.Align`; switch the
  alignment-forcing element to authoritative align. Invert the union assertion in
  `scanner_test.go`.
- Gate: re-scan; metadata diff shows union entries + size/align; parity = 0.

**Commit 3 — union emission via byte-array + accessors (idiomatic + raw together).**
- Emit named unions and the 6 embedded-union sized structs; all union members at
  offset 0. Raw gets the same aligned-`[N]byte` body (ABI-faithful; today raw
  degrades union fields to `unsafe.Pointer`, wrong size).
- Gate: parity = 0; metal build confirms `MTLPackedFloat3` is 12 bytes; reflect test
  extended to unions.

**Commit 4 (optional, later) — fn-ptr fields.** Needs a scanner change to size
pointer-only dispatch structs + the `RegisterFunc` trampoline accessor. After 1–3.

Only Commits 2 and 4 need a scanner change / re-scan.

## Verify (this repo)
```sh
go build $(go list ./... | grep -v /internal/acceptance)   # codegen
go run ./cmd/generate/ idiomatic && go run ./cmd/generate/ bindings
go run ./cmd/generate/ parity        # 0 missing, both pipelines
go test ./internal/codegen/... ./internal/scanner/...
# determinism: regen twice → byte-identical
```
