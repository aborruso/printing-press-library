.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/openrouter-pp-cli$(BIN_EXT) ./cmd/openrouter-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/openrouter-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/openrouter-pp-mcp$(BIN_EXT) ./cmd/openrouter-pp-mcp

install-mcp:
	go install ./cmd/openrouter-pp-mcp

build-all: build build-mcp
