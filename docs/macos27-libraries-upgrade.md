# macOS 27 C-library refresh

This follow-up starts from main commit `52a268a0a632`, after framework PR
[#129](https://github.com/deploymenttheory/go-bindings-macosplatform/pull/129)
merged. It refreshes all 15 registered C libraries and regenerates their raw and
public layers. The public library tree still contains 16 packages, including the
unchanged BSD support package. Framework metadata and generated framework
bindings remain unchanged.

## Provenance and API changes

- Xcode 27.0 build `27A266a`, SDK 27.0 arm64, Apple Clang 21.0.0
  (`clang-2100.3.34.2`). All 15 libraries were rescanned successfully, with one
  SDK 27.0 metadata artifact per library.
- Local validation uses macOS 27.0 build `26A428` and Go 1.27.1. The minimum
  runtime remains macOS 27; Go module dependencies and the CI runner matrix are
  unchanged.
- The API comparison finds changes in 11 libraries, with no library additions
  or removals. See [the library API diff](macos27-libraries-api-diff.md).
- EndpointSecurity adds nine functions, three structs, new bootstrap/XPC events,
  and deadline-miss controls. Compression and AppleArchive add compression
  algorithm constants. Inherited Mach enum declarations are also refreshed.
- Sandbox removes the five legacy `kSBXProfile*` externs. They are absent from
  the new SDK headers and are removed from both binding layers.
- Three XPC handler declarations now resolve to Go callbacks instead of
  `unsafe.Pointer`. The public emitter preserves their method/function names
  and forwards the typed callbacks through the raw block adapter. Callers of
  `Connection.SetEventHandler`, `Connection.SendMessageWithReply`, and
  `SetEventStreamHandler` must now supply `func(unsafe.Pointer)` handlers.

The library runtime now honors the scanner's `link_name`: dispatch queue
creation with a target and its two queue assertions bind their `$V2` symbols.
Go wrapper names and `SymbolAvailable` keys retain the C declaration names.
Emitter regression tests cover both aliased and ordinary exports; a native
queue test exercises the versioned functions through the public bindings.

## EndpointSecurity ABI corrections

Two SDK 27 unions need inline storage. Mapping them to `unsafe.Pointer` would
misrepresent the bootstrap target and place `es_process_t.cdhash_full` at the
wrong offset. Scoped [metadata overrides](../metadata/libraries/endpointsecurity/overrides.json)
map them to C arrays of the measured size and alignment. The scanned metadata
remains untouched. Shared override support now permits named or unambiguous
anonymous struct fields; both pipeline loaders use the same implementation.

Native `sizeof`, `_Alignof`, and `offsetof` probes against the installed SDK
establish the following arm64 layout, checked by pure-Go acceptance tests:

| Record or field | Size | Alignment or offset |
|---|---:|---:|
| `es_event_bootstrap_check_in_t` | 56 | alignment 8 |
| `es_event_bootstrap_look_up_t` | 120 | alignment 8 |
| `es_lightweight_code_requirement_t` | 32 | alignment 8 |
| `es_process_t` | 248 | alignment 8 |
| Bootstrap `target` union | 56 | offset 64, alignment 8 |
| Process anonymous `reserved` union | 16 | offset 212, alignment 1 |
| Process `cdhash_full` | 16 | offset 232 |

The union storage is intentionally opaque; this change does not add typed
accessors for each union arm. Other existing library struct-mapping limitations
remain outside this SDK refresh; the new layout tests cover the records above.

## Documentation and diagnostics

All 15 library DocC sidecars were refreshed using a new online cache: 205
published pages, 390 missing pages, and zero fetch errors. Missing documentation
continues to use header comments. Framework documentation is unchanged.

The diagnostics baseline remains at 2,775 entries. Nineteen EndpointSecurity
entries move with their header locations; after normalizing those locations,
the diagnostic sets are identical. The two newly encountered union mappings are
corrected by the ABI overrides rather than accepted as pointer degradations.
Metadata validation reports zero errors and 366 existing warnings; the 15
mixed-SDK warnings from the framework-only phase disappear.

## Validation

Validation results are recorded after the final regeneration and checks.
