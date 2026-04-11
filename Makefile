.PHONY: help check-go-version setup-go-gvm checks build test e2e-url-only lint vet pre-commit-install pre-commit-run clean

.DEFAULT_GOAL := build

BINARY := qrrun
CMD     := ./cmd/qrrun
GOLANGCI_LINT_VERSION := v1.64.8
GO_REQUIRED_VERSION := 1.24.13
VERSION ?= v0.0.0-dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

## help: show available make targets
help:
	@awk '/^## / {desc=substr($$0, 4); next} /^[a-zA-Z0-9_.-]+:/ {if (desc != "") {split($$1, t, ":"); printf "%-16s %s\n", t[1], desc; desc=""}}' $(MAKEFILE_LIST)

## check-go-version: verify the active Go version matches project requirement
check-go-version:
	@go version | grep -q "go$(GO_REQUIRED_VERSION)" || { echo "Expected Go $(GO_REQUIRED_VERSION). Run 'make setup-go-gvm' if you use gvm."; exit 1; }

## setup-go-gvm: install and activate the required Go version via gvm
setup-go-gvm:
	@test -f "$$HOME/.gvm/scripts/gvm" || { echo "gvm is not installed. See README for install steps."; exit 1; }
	@bash --noprofile --norc -lc 'source "$$HOME/.gvm/scripts/gvm" && gvm list | grep -q "go$(GO_REQUIRED_VERSION)" || gvm install go$(GO_REQUIRED_VERSION)'
	@bash --noprofile --norc -lc 'source "$$HOME/.gvm/scripts/gvm" && gvm use go$(GO_REQUIRED_VERSION) --default'

## build: compile the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## test: run all tests
test:
	go test ./...

## checks: run CI-like local checks (test, vet, build, lint)
checks: test vet build lint

## e2e-url-only: run local --url-only end-to-end test via cloudflared
e2e-url-only: build
	bash ./scripts/e2e_url_only.sh

## lint: run golangci-lint (uses go run fallback when not installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		go run github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...; \
	fi

## vet: run go vet
vet:
	go vet ./...

## pre-commit-install: install git pre-commit hooks
pre-commit-install:
	pre-commit install

## pre-commit-run: run all pre-commit hooks across all files
pre-commit-run:
	pre-commit run --all-files

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
