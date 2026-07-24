# codegen package scope convention

The code generator has **two pipelines** that are intentionally kept separate —
this split is a non-negotiable of the architecture:

| Pipeline | Binds via | Struct ABI | Packages |
|---|---|---|---|
| **frameworks** | purego / ObjC runtime | Go struct passed **by value** through purego func vars → Go layout must match the C ABI byte-exactly | `frameworks/*`, `emit/*/frameworks` |
| **libraries** | cgo / C bridge shims | the C bridge handles the ABI | `libraries/*`, `emit/*/libraries` |

Because the two pipelines bind differently, their type mappers, naming, and
emitters **diverge in behaviour, not just structure**. Do not assume that a
function existing on both sides is shareable — many same-named helpers
(`IsCoreFoundationOpaqueRef`, `normalise`/`Normalise`, `splitCSV`/`splitArgs`, the
block/fn-ptr parsers, the primitive tables) have different implementations and
different outputs on purpose. Merging them would change generated code.

## The `SCOPE —` convention

Every non-obvious package declares its scope in its package doc with a scannable
`SCOPE —` line. Run `grep -rn "SCOPE —" internal/codegen` for the live map.

- **SHARED** — pipeline-agnostic, used by both. Keep these free of either
  pipeline's mapper/metadata:
  - `emit/structlayout` — ABI/layout reasoning (size, alignment, byte-array tier)
  - `emit/layouttest` — the generated ABI layout regression test
  - `emitmanifest` — the parity manifest recorded by both generator styles
  - `shared/fileasm` — generated-file scaffolding
  - `pipeline/structindex` — struct-ownership index for both loaders
  - `naming/core` — only behaviour-identical naming helpers
- **FRAMEWORKS pipeline (purego/ObjC)** — everything under `frameworks/` and
  `emit/*/frameworks`. `frameworks/typemap` is pipeline-specific by design.
- **LIBRARIES pipeline (cgo/C)** — everything under `libraries/` and
  `emit/*/libraries`. `libraries/typemap` is pipeline-specific by design and
  additionally carries a `CType()` dimension the frameworks side has no use for.

## Rule of thumb

Shared logic must be **behaviour-identical** across both pipelines; the moment a
helper needs to differ, it belongs in the pipeline package, not in a shared one.
A path segment (`frameworks/`, `libraries/`) or a `SCOPE —` line should make every
package's allegiance obvious without reading its body.
