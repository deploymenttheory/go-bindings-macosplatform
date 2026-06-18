# Apple developer documentation enrichment

Generated bindings carry documentation from two sources, merged at load time:

1. **Header doc comments** — captured by the scanner (`internal/scanner/comments.go`,
   `docForNode`) into the `Doc` field of every symbol in the committed
   `.gometa.json`. These are whatever Doxygen/`///` comments ship in the SDK headers
   and are often sparse.
2. **Apple's web documentation** — Apple's richer abstracts (and, in `--deep` mode,
   Discussions) harvested from the DocC render API by the
   [`scripts/tools/appledeveloperdocs`](../scripts/tools/appledeveloperdocs) tool into
   per-framework `appledocs.json` sidecars.

## Sidecar files

The sidecar sits next to the `.gometa.json` it enriches — the same adjacency
convention as `overrides.json`:

```
metadata/frameworks/<name>/appledocs.json
metadata/libraries/<name>/appledocs.json
```

It is keyed by metadata identity: class/enum/struct/protocol names, the `±selector`
method key (`-`/`+` prefix, see `appledocs.MethodKey`), property names, and enum
member names. Schema and discovery live in
[`internal/appledocs`](../internal/appledocs/appledocs.go).

The committed `.gometa.json` stays **pure scanned Clang output** — Apple docs are a
separate, regenerable layer, so an SDK re-scan never clobbers them.

## Merge at load time

Both pipeline loaders apply the sidecar right after `overrides`, mirroring that
package exactly:

- libraries (CGo): `internal/codegen/libraries/pipeline/loader.go` →
  `internal/appledocs.ApplyAdjacent`
- frameworks (purego): `internal/codegen/frameworks/pipeline/loader.go` →
  `internal/codegen/frameworks/appledocs.ApplyAdjacent`

Merge policy is **Apple-preferred, header fallback**: a non-empty Apple abstract
overwrites the header `Doc`; otherwise the header comment is kept. The raw emitters
already render `Doc`, so they need no change. The idiomatic emitter now renders
`Doc` too (a `DocComment` block prepended above each wrapper type, constructor,
setter, and method).

## Regenerating

```sh
# 1. Harvest (network). Writes metadata/.../appledocs.json sidecars.
go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation

# 2. Regenerate bindings + idiomatic from the enriched metadata (no network).
go run ./cmd/generate/ bindings
go run ./cmd/generate/ idiomatic
```

See the [tool README](../scripts/tools/appledeveloperdocs/README.md) for fetch
flags and the DocC/Objective-C-projection details.
