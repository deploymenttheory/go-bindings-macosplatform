ACC_N    ?= 100
ACC_SEED ?= 0
ACC_ATTEST ?= acceptance-attestation.jsonl
# Consumer project rebuilt against the local tree by canary-build.
CANARY_DIR ?= ../go-macos-observability

.PHONY: help generate build test lint version acc-generate acc-test act-acc regen-diff canary-build parity parity-update

help:
	@echo "Targets:"
	@echo "  generate      Re-emit all bindings from committed metadata (no Clang scan)"
	@echo "  build         Build all binding packages, opinionated tools, and examples"
	@echo "  test          Run internal unit tests"
	@echo "  purego-test   Live tests for purego-backed C libraries with CGO_ENABLED=0"
	@echo "  lint          Run golangci-lint"
	@echo "  version       Detect macOS/SDK versions and check acceptance test compatibility"
	@echo "  acc-generate  Generate the dynamic acceptance test file (ACC_N, ACC_SEED)"
	@echo "  acc-test      Generate and run acceptance tests locally"
	@echo "  act-acc       Run the acceptance workflow via act (requires act in PATH)"
	@echo "  regen-diff    Re-emit raw + idiomatic bindings, build+vet, and show the diff"
	@echo "  canary-build  Build the consumer project in CANARY_DIR against this working tree"
	@echo "  parity        Check idiomatic emittance covers the raw oracle (ratchet vs committed baseline)"
	@echo "  parity-update Rewrite the parity baseline after a phase closes gaps"

generate:
	go run ./cmd/generate/ bindings

build:
	go build ./bindings/... ./opinionated/... ./examples/...

test:
	go test ./internal/...

# Live acceptance for the purego-backed C libraries, with cgo disabled — the
# suite compiling at all proves the migrated libraries are pure Go.
purego-test:
	CGO_ENABLED=0 go test ./bindings/acceptance/puregolibs/ -count=1 -v

lint:
	golangci-lint run --timeout 30m

version:
	go run ./scripts/ci/macosversion/

acc-generate:
	go run ./cmd/genacceptance/ --n $(ACC_N) --seed $(ACC_SEED)

acc-test: acc-generate
	GENACCEPT_ATTEST=$(ACC_ATTEST) \
	go test ./bindings/acceptance/ -v -timeout 600s -count=1

act-acc:
	act workflow_dispatch \
	  -W .github/workflows/acceptance.yml \
	  --input n=$(ACC_N) \
	  --input seed=$(ACC_SEED)

# Regenerate raw + idiomatic bindings from committed metadata, prove they still
# build/vet, then show what changed. The diff is the review artifact for every
# emitter change: an unexpected hunk means an unintended behavior change. Raw
# lands under bindings/internal/raw; the public idiomatic layer under
# bindings/{frameworks,libraries}.
regen-diff:
	go run ./cmd/generate/ bindings
	go run ./cmd/generate/ idiomatic
	go build ./bindings/...
	# -unsafeptr=false: the generated extern accessors dereference dlsym
	# addresses (uintptr → unsafe.Pointer), a known-safe FFI pattern that the
	# unsafeptr check cannot prove.
	go vet -unsafeptr=false ./bindings/...
	git --no-pager diff --stat -- bindings/
	@echo "Run 'git diff -- bindings/' to review the full diff."

# Prove the idiomatic emitter covers every construct the raw emitter does. The
# ratchet fails only on a NEW gap not already in the committed baseline, so it
# passes during the migration while gaps are being closed phase by phase, and
# guards against regressions. Shrink the baseline with parity-update as gaps close.
parity:
	go run ./cmd/generate/ parity --baseline metadata/parity-baseline.json

parity-update:
	go run ./cmd/generate/ parity --write-baseline metadata/parity-baseline.json

# Compile the consumer project against this working tree (no go.mod edits in
# either repo: a throwaway workspace file under the gitignored tmp/ wires the
# two modules together).
canary-build:
	@mkdir -p tmp
	@printf 'go 1.26.2\n\nuse (\n\t%s\n\t%s\n)\n' "$(CURDIR)" "$(abspath $(CANARY_DIR))" > tmp/canary.work
	cd $(CANARY_DIR) && GOWORK=$(CURDIR)/tmp/canary.work go build ./...
