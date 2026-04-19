.PHONY: help all build test test-e2e lint vet pre-commit-install pre-commit-run clean

.DEFAULT_GOAL := build

BINARY := qrrun
CMD     := ./cmd/qrrun
GOLANGCI_LINT_VERSION := v1.64.8
VERSION ?= v0.0.0-dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

## help: show available make targets
help:
	@awk '/^## / {desc=substr($$0, 4); next} /^[a-zA-Z0-9_.-]+:/ {if (desc != "") {split($$1, t, ":"); printf "%-16s %s\n", t[1], desc; desc=""}}' $(MAKEFILE_LIST)

## all: default aggregate target
all: build

## build: compile the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## test: run all tests
test:
	go test ./...

## test-e2e: run local --print-url end-to-end test via cloudflared
test-e2e: build
	python3 -m unittest discover -v -s test.e2e -p "test_*.py"

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
