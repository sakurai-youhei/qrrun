.PHONY: help build test lint vet clean

BINARY := qrrun
CMD     := ./cmd/qrrun

## help: show available make targets
help:
	@awk '/^## / {desc=substr($$0, 4); next} /^[a-zA-Z0-9_.-]+:/ {if (desc != "") {split($$1, t, ":"); printf "%-16s %s\n", t[1], desc; desc=""}}' $(MAKEFILE_LIST)

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
