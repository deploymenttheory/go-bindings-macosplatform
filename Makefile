ACC_N    ?= 100
ACC_SEED ?= 0
ACC_ATTEST ?= acceptance-attestation.jsonl
# Consumer project rebuilt against the local tree by canary-build.
CANARY_DIR ?= ../go-macos-observability

.PHONY: help generate build test lint version acc-generate acc-test act-acc idiomatic-regen-diff canary-build

help:
	@echo "Targets:"
	@echo "  generate      Re-emit all bindings from committed metadata (no Clang scan)"
	@echo "  build         Build all generated binding packages"
	@echo "  test          Run internal unit tests"
	@echo "  lint          Run golangci-lint"
	@echo "  version       Detect macOS/SDK versions and check acceptance test compatibility"
	@echo "  acc-generate  Generate the dynamic acceptance test file (ACC_N, ACC_SEED)"
	@echo "  acc-test      Generate and run acceptance tests locally"
	@echo "  act-acc       Run the acceptance workflow via act (requires act in PATH)"
	@echo "  idiomatic-regen-diff  Re-emit the idiomatic layer, build+vet it, and show the diff"
	@echo "  canary-build  Build the consumer project in CANARY_DIR against this working tree"

generate:
	go run ./cmd/generate/ bindings

build:
	go build ./bindings/...

test:
	go test ./internal/...

lint:
	golangci-lint run --timeout 30m

version:
	go run ./scripts/ci/macosversion/

acc-generate:
	go run ./cmd/genacceptance/ --n $(ACC_N) --seed $(ACC_SEED)

acc-test: acc-generate
	GENACCEPT_ATTEST=$(ACC_ATTEST) \
	go test ./acceptance/ -v -timeout 600s -count=1

act-acc:
	act workflow_dispatch \
	  -W .github/workflows/acceptance.yml \
	  --input n=$(ACC_N) \
	  --input seed=$(ACC_SEED)

# Regenerate the idiomatic layer from committed metadata, prove it still
# builds/vets, then show what changed. The diff is the review artifact for
# every emitter change: an unexpected hunk means an unintended behavior change.
idiomatic-regen-diff:
	go run ./cmd/generate/ idiomatic
	go build ./opinionated/...
	# -unsafeptr=false: the generated extern accessors dereference dlsym
	# addresses (uintptr → unsafe.Pointer), a known-safe FFI pattern that the
	# unsafeptr check cannot prove.
	go vet -unsafeptr=false ./opinionated/...
	git --no-pager diff --stat -- opinionated/
	@echo "Run 'git diff -- opinionated/' to review the full diff."

# Compile the consumer project against this working tree (no go.mod edits in
# either repo: a throwaway workspace file under the gitignored tmp/ wires the
# two modules together).
canary-build:
	@mkdir -p tmp
	@printf 'go 1.26.2\n\nuse (\n\t%s\n\t%s\n)\n' "$(CURDIR)" "$(abspath $(CANARY_DIR))" > tmp/canary.work
	cd $(CANARY_DIR) && GOWORK=$(CURDIR)/tmp/canary.work go build ./...
