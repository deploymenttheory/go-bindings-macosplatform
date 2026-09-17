# macOS 27 SDK upgrade

This upgrade raises the minimum supported runtime to macOS 27 and regenerates
both framework binding layers from the arm64 macOS 27.0 SDK. It adds eight
framework entries,
bringing the public tree to 260 framework packages and 16 C-library packages.
The C-library metadata and bindings remain at their existing SDK 26.5 revision.
Their SDK 27 refresh will be a separate PR, started from main after this framework
PR is merged. Swift-only entries retain the generator's existing stub behavior.

## Provenance

- Previous metadata: SDK 26.5, repository commit `bdaa456054b8`.
- Xcode 27.0, build `27A266a`; Apple Clang 21.0.0 (`clang-2100.3.34.2`).
- SDK 27.0, arm64; local validation on macOS 27.0 build `26A428` with Go 1.27.1.
- All 268 framework entries were scanned successfully, including IOKit and
  IOKit/PowerSources and their record layouts. Each framework has one SDK 27.0
  arm64 artifact. The 15 existing C-library metadata entries stay on SDK 26.5.
- Consumer builds remain pure Go. Xcode is needed for SDK extraction, not for
  building applications from these bindings. Module dependencies are unchanged.

## API changes and generator corrections

The semantic comparison records eight added framework entries, no removed
frameworks, 269 added classes, seven removed classes, 651 added methods,
28 removed methods, and 279 changed method signatures. Signature changes also
include corrections to previously malformed annotated types.

The additions are AccessoryAccess, AppIntentsTypeSupport, ComputeGraph,
LinkSecurity, MPSFunctions, SpatialPreview, StateReporting, and
`_ScreenCaptureKit_SwiftUI`. The full comparison is in
[the API diff](macos27-api-diff.md).

The seven removed classes belong to Matter's former Timer API. Header review also
confirmed removals in Matter's Groupcast APIs, two DirectoryService functions,
AVKit's legible-menu enum, legacy Sandbox profile constants, PhotosUI constructors,
and ParavirtualizedGraphics version constants. ParavirtualizedGraphics APIs
marked obsoleted in macOS 27 are now excluded from emitted methods.

Changes needed to extract and bind this SDK correctly:

- ARKit now forwards to private headers unless `USE_ARKIT_PUBLIC_HEADERS=1` is
  defined. The scan configuration supplies it to both Clang passes, preserving
  the public C surface and its ABI layouts.
- The scanner removes stacked availability/ownership annotations from declaration
  types and preserves both forms of macOS obsoletion annotations.
- Realtime audio block parsing handles trailing `nonblocking` attributes.
  Const-qualified scalars, typedefs, and array elements retain their value types.
- Enum overflow detection compares actual numeric values. GameController's
  signed `NSIntegerMax` and negative defaults remain signed.
- Generated filenames for underscore-prefixed frameworks now start with `sdk`,
  making their declarations visible to Go without changing package paths.
- C function assembly aliases are recorded as `link_name` and used for symbol
  lookup in both layers and the symbol gate. Go API names stay stable. This fixes
  SDK 27 BNNS declarations that redirect to `_v2` exports, as well as
  three Security aliases. Library-emitter changes are deferred to the follow-up.

C-string extern accessors now dereference the exported pointer before decoding
it through the runtime helper. The vmnet acceptance test covers this without
requiring network entitlements. Raw C functions now pass protocol-qualified
objects as `objc.ID` handles (and pointers to handles for out-parameters),
avoiding unsupported Go interfaces in native call signatures.

Regression tests exercise real Clang header selection, layouts and assembly
aliases, generated enum type checking, Go source-file discovery, annotated types,
block parsing, and const values.

## Documentation and isolation

Apple documentation was refreshed from the online DocC API with a fresh cache:
all supported framework sidecars, with zero fetch errors. The complete harvest
visited 9,319 published pages and 14,910 missing pages; only framework sidecars
are included in this PR.
Missing published pages retain the header-comment fallback. ARKit was refreshed
again after correcting public-header selection. Seven Swift-only entries do not
participate in this harvest.

Swift isolation extraction completed for 216 entries. It added AppKit's
NSRefreshController and NSTextSelectionManager, three new WebKit classes,
AVSampleBufferDisplayLayer, and AVAudioPlayerDelegate; no existing class or
protocol isolation facts were removed. There were 45 extraction failures,
primarily subframeworks without standalone Swift modules and legacy frameworks
that reject Swift import. None had a previous isolation sidecar to lose; these
remain a coverage limitation of the extraction tool.

## Baselines and validation

Metadata validation reports zero errors. The framework scan has 366 integrity
warnings, down from 382; retaining the old C-library metadata adds 15 expected
SDK-consistency warnings until the separate library upgrade. Five
new ownership-tie warnings arise from MPSFunctions sharing declarations with its
umbrella framework; ownership remains deterministic. The other warnings are
existing metadata debt.

The reviewed type-degradation changes are listed individually in
[the degradation review](macos27-degradations.md). The baseline falls from 3,140
to 2,775 entries (56 reviewed additions and 421 removals). C-library diagnostics
remain at their original SDK revision. Anonymous unions still use
measured byte layouts; opaque handles remain pointers, and unsupported
object-returning callbacks retain explicit `objc.Block` representations.

Validation completed on macOS 27.0 arm64:

- Generator/tool unit tests pass.
- The complete bindings, opinionated tools, and examples build with
  `CGO_ENABLED=0`; binding vet checks pass. Both include underscore-prefixed
  framework packages explicitly.
- All 86 generated ABI layout suites pass across the public and raw layers.
- Runtime and consumer tests pass, as do all 23 purego library smoke tests with
  `CGO_ENABLED=0` against the unchanged C-library bindings.
- Metadata validation: zero errors; 381 warnings, including the 15 deliberately
  deferred library SDK versions.
- Parity: 147,098 raw framework constructs, 147,107 public constructs, zero missing;
  unchanged libraries cover all 1,866 raw exports.
- Symbol gate: 13,159 function symbols, 12,690 externs and 5,174 class entries pass,
  with no additions to the unresolved-symbol baseline.
- The reported runtime-read and vmnet suites pass: 103 tests and one existing
  opt-in skip for privileged vmnet networking.
- The full acceptance run passes 239 tests, including 100 generated samples with
  hexadecimal seed `2700`. Two existing skips remain: NSNumber negative sign
  extension and privileged vmnet networking. All 135 emitted attestation records
  report success.
- Full-file Go lint passes with zero issues in the generator/tool scope. Three
  existing scanner state/dispatch functions retain scoped complexity exemptions;
  formatting and error-style findings in touched files were corrected.
- Acceptance workflow lint passes. Its attestation path is absolute so the
  artifact upload collects the file written by the test package.
- A second hierarchy/raw/public regeneration passes the strict diagnostics
  baseline and produces identical hashes for all 14,392 generated Go files.
  C-library metadata, bindings, and emitter code have no diff against main.

CI retains `os: [macos-latest, xcode-27]`. Both jobs perform static checks; live
checks run on macOS 27+. The xcode-27 job fails if its runtime is not macOS 27,
so a toolchain label cannot silently replace live runtime coverage.

See [the extraction workflow](extraction_workflow.md) for reproducible commands.
