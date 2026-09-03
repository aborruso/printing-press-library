.PHONY: build test lint install clean

build:
	go build -o bin/pixabay-pp-cli ./cmd/pixabay-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/pixabay-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/pixabay-pp-mcp ./cmd/pixabay-pp-mcp

install-mcp:
	go install ./cmd/pixabay-pp-mcp

build-all: build build-mcp
