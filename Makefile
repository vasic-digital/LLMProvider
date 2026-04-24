.PHONY: build test test-core test-providers vet clean help

## Build all packages
build:
	go build ./...

## Run all tests with race detector
test:
	go test ./... -race -count=1

## Run core tests only (no network calls)
test-core:
	go test ./pkg/models/... ./pkg/retry/... ./pkg/circuit/... ./pkg/health/... ./pkg/provider/... ./pkg/http/... -race -count=1

## Run provider tests
test-providers:
	go test ./pkg/providers/... -race -count=1

## Run vet
vet:
	go vet ./...

## Clean build cache
clean:
	go clean -cache -testcache

## Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build all packages"
	@echo "  test           - Run all tests with race detector"
	@echo "  test-core      - Run core tests only (no network calls)"
	@echo "  test-providers - Run provider tests"
	@echo "  vet            - Run go vet"
	@echo "  clean          - Clean build cache"

# Definition of Done gates — portable drop-in from HelixAgent
.PHONY: no-silent-skips no-silent-skips-warn demo-all demo-all-warn demo-one ci-validate-all

no-silent-skips:
	@bash scripts/no-silent-skips.sh

no-silent-skips-warn:
	@NO_SILENT_SKIPS_WARN_ONLY=1 bash scripts/no-silent-skips.sh

demo-all:
	@bash scripts/demo-all.sh

demo-all-warn:
	@DEMO_ALL_WARN_ONLY=1 DEMO_ALLOW_TODO=1 bash scripts/demo-all.sh

demo-one:
	@DEMO_MODULES="$(MOD)" bash scripts/demo-all.sh

ci-validate-all: no-silent-skips-warn demo-all-warn
	@echo "ci-validate-all: all gates executed"
