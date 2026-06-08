.PHONY: build test lint install clean

build:
	go build -o bin/verto-pp-cli ./cmd/verto-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/verto-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/verto-pp-mcp ./cmd/verto-pp-mcp

install-mcp:
	go install ./cmd/verto-pp-mcp

build-all: build build-mcp
