.PHONY: help gvm-setup build test lint vet clean

BINARY := qrrun
CMD     := ./cmd/qrrun

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
	go build -o $(BINARY) $(CMD)

## test: run all tests
test:
	go test ./...

## lint: run golangci-lint (requires golangci-lint to be installed)
lint:
	golangci-lint run ./...

## vet: run go vet
vet:
	go vet ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
