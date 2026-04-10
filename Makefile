.PHONY: build test lint clean

BINARY := qrrun
CMD     := ./cmd/qrrun

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
