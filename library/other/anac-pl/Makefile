.PHONY: build test lint install clean

build:
	go build -o bin/anac-pl-pp-cli ./cmd/anac-pl-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/anac-pl-pp-cli

clean:
	rm -rf bin/
	rm -f anac-pl-pp-cli anac-pl-pp-mcp anac-pl-pp-cli-dogfood

build-mcp:
	go build -o bin/anac-pl-pp-mcp ./cmd/anac-pl-pp-mcp

install-mcp:
	go install ./cmd/anac-pl-pp-mcp

build-all: build build-mcp
