.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/listingview-pp-cli$(BIN_EXT) ./cmd/listingview-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/listingview-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/listingview-pp-mcp$(BIN_EXT) ./cmd/listingview-pp-mcp

install-mcp:
	go install ./cmd/listingview-pp-mcp

build-all: build build-mcp
