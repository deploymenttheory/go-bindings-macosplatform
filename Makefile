ACC_N    ?= 100
ACC_SEED ?= 0
ACC_ATTEST ?= acceptance-attestation.jsonl

.PHONY: help generate build test lint version acc-generate acc-test act-acc

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
