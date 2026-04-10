.PHONY: help gvm-setup build test lint vet clean

BINARY := qrrun
CMD     := ./cmd/qrrun
GOLANGCI_LINT_VERSION := v1.64.8
VERSION ?= v0.1.0-alpha.0
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

## help: show available make targets
help:
	@awk '/^## / {desc=substr($$0, 4); next} /^[a-zA-Z0-9_.-]+:/ {if (desc != "") {split($$1, t, ":"); printf "%-16s %s\n", t[1], desc; desc=""}}' $(MAKEFILE_LIST)

## gvm-setup: install and use the Go version required by this project
gvm-setup:
	@test -f "$$HOME/.gvm/scripts/gvm" || { echo "gvm is not installed. See README for install steps."; exit 1; }
	@bash --noprofile --norc -lc 'source "$$HOME/.gvm/scripts/gvm" && gvm list | grep -q "go1.24.13" || gvm install go1.24.13'
	@bash --noprofile --norc -lc 'source "$$HOME/.gvm/scripts/gvm" && gvm use go1.24.13 --default'

## build: compile the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## test: run all tests
test:
	go test ./...

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

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
