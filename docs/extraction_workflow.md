# Extraction and SDK upgrade workflow

The project generates macOS 27+ Go bindings from the macOS 27.0 SDK shipped
with Xcode 27.0. Both Objective-C frameworks and Apple C libraries use purego;
consumer builds and regeneration from committed metadata need no C compiler.
Xcode is required for scanning headers and extracting Swift main-thread isolation.

## Pipeline

```text
Xcode SDK headers → Clang AST + record layouts → .gometa.json
                                               ↓
                 overrides + Apple docs + main-thread isolation
                                               ↓
                    independent framework/library registries
                                               ↓
                 resolved view models → template rendering
                                               ↓
           internal raw bindings + public idiomatic bindings
```

The scanner extracts classes, protocols, enums, structs, C functions, externs,
block signatures, availability, and header comments. A second Clang pass supplies
record sizes, alignments, and field offsets. When Clang omits availability values
from its JSON AST, the scanner reads the macros at the declaration's source location.

Metadata is committed under `metadata/frameworks/<name>/` and
`metadata/libraries/<name>/`. Sub-frameworks use another directory level, such as
`metadata/frameworks/iokit/powersources/`. Each file records the SDK version,
architecture, schema version, and Clang/Xcode provenance.

The framework and library loaders maintain separate type mappers and ownership
indices. Overrides correct local metadata defects without modifying scanned
output. Apple documentation and `@MainActor` sidecars enrich the loaded model.
The emitters resolve types into view models and render Go through templates.

- `generate bindings` writes the internal raw mirror under `bindings/internal/raw/`.
- `generate idiomatic` writes the public framework/library APIs and generated support.
- Public frameworks dispatch directly through the runtime; public libraries wrap
  their raw counterparts and re-export value types. External consumers cannot
  import the raw layer.
- Swift-only surfaces retain their existing type-only or documentation-stub behavior.

## Everyday regeneration

Run from the repository root:

```sh
go run ./cmd/generate/ bindings
go run ./cmd/generate/ idiomatic
go run ./cmd/generate/ validate
go run ./cmd/generate/ parity --baseline metadata/parity-baseline.json
```

Run both emission steps after changing a generator. `make generate` currently
runs only the raw step; `make regen-diff` runs both, builds/vets, and displays the
generated diff. Never repair a generated declaration by editing the output.

## Upgrading the SDK

1. Verify `xcodebuild -version`, `xcrun --show-sdk-version`,
   `xcrun --show-sdk-path`, and `xcrun clang --version`. Record the exact toolchain.
2. Preserve the old metadata tree and validation output for comparison. Create
   `tmp/macos27/metadata` containing the existing configuration and sidecars but
   **no old `.gometa.json` files**. Keep the committed metadata intact during staging.
3. Scan every discoverable framework and registered library, then the custom
   IOKit headers that standard umbrella-header discovery does not include:

   ```sh
   go run ./cmd/generate/ scan --framework all \
     --metadata-dir tmp/macos27/metadata --arch arm64 --parallel 4
   go run ./cmd/generate/ scan --framework IOKit,IOKit/PowerSources \
     --metadata-dir tmp/macos27/metadata --arch arm64 --parallel 4
   ```

4. Reconcile successful scans against the SDK inventory and existing coverage.
   Investigate every failed scan, unexpected loss, and record-layout fallback.
   Require one current arm64 metadata file per entry, with matching SDK provenance.
   **Do not leave old and new SDK files together:** loaders prefer arm64 but do
   not select the newest SDK version within an architecture.
5. Refresh the sidecars against staged metadata:

   ```sh
   go run ./scripts/tools/mainactorisolation fetch --framework all \
     --metadata tmp/macos27/metadata
   go run ./scripts/tools/appledeveloperdocs fetch --framework all \
     --metadata tmp/macos27/metadata --cache tmp/macos27/docc-cache
   ```

   Use a new DocC cache for each upgrade; cached 404s otherwise hide newly
   published pages. The default harvest fetches abstracts. Audit warnings and
   error counts: a successful process exit alone does not prove every framework
   or documentation page refreshed. A missing published page uses header docs.
   Main-thread extraction failures need individual review, especially for UI APIs.
6. Validate and compare before replacing committed metadata:

   ```sh
   go run ./cmd/generate/ validate --metadata-dir tmp/macos27/metadata
   go run ./cmd/generate/ diff --old tmp/macos27/baseline/metadata \
     --new tmp/macos27/metadata
   ```

   Use `--json` for machine-readable API changes. Review added/removed declarations,
   changed signatures and availability, and new integrity warnings. Preserve valid
   overrides and configuration when promoting the complete staged tree.
7. Regenerate the hierarchy, raw mirror, and public API in order:

   ```sh
   go run ./cmd/generate/ class-hierarchy
   go run ./cmd/generate/ bindings --diagnostics-baseline metadata/diagnostics-baseline.json
   go run ./cmd/generate/ idiomatic
   ```

   Resolve SDK changes in scanner/mapper/emitter code or scoped metadata overrides.
   Examine new degradation or unresolved-symbol entries individually; explain any
   accepted baseline change in the upgrade report. Keep parity at zero missing.
8. Update runtime requirements, coverage counts, and CI's actual runtime check.
   Keep the matrix `os: [macos-latest, xcode-27]`. Both runners build and check
   generation; live tests run on macOS 27+. The `xcode-27` job requires macOS 27.
   A toolchain label alone must not silently allow live tests to be skipped.

## Verification

```sh
go test ./internal/... ./scripts/tools/... ./scripts/ci/...
go run ./cmd/generate/ validate
go run ./cmd/generate/ parity --baseline metadata/parity-baseline.json
CGO_ENABLED=0 go build ./bindings/... ./bindings/frameworks/_* ./bindings/internal/raw/frameworks/_* ./opinionated/... ./examples/...
go vet -unsafeptr=false ./bindings/... ./bindings/frameworks/_* ./bindings/internal/raw/frameworks/_*
go test ./bindings/frameworks/... ./bindings/internal/raw/frameworks/... ./bindings/frameworks/_* ./bindings/internal/raw/frameworks/_* -run '^TestGeneratedLayout$' -count=1
go test ./bindings/runtime/... ./opinionated/... ./examples/... -count=1
CGO_ENABLED=0 go test ./bindings/acceptance/puregolibs/ -count=1 -v
go run ./cmd/gensymbolgate/
go test ./bindings/symbolgate/ -v -timeout 300s -count=1
go run ./cmd/genacceptance/ --n 100 --seed 2700
GENACCEPT_ATTEST="$PWD/acceptance-attestation.jsonl" go test ./bindings/acceptance/ -v -timeout 600s -count=1
```

The fixed hexadecimal seed makes the sampled corpus reproducible. Record existing
intentional skips separately from failures. The symbol gate checks resolution,
layout tests check ABI sizes/alignment, and live tests check real call results.
They cover different failure modes.

Finally regenerate both layers again from the same metadata and compare file
hashes; the second emission must produce identical bytes. Preserve the semantic
API diff, exact toolchain, baseline changes, and test results in the upgrade report.
