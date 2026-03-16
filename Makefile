MODULE  := github.com/iminders/aicoder
BINARY  := aicoder
VERSION ?= 1.0.0
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X $(MODULE)/pkg/version.Version=$(VERSION) \
           -X $(MODULE)/pkg/version.Commit=$(COMMIT)   \
           -X $(MODULE)/pkg/version.BuildDate=$(DATE)

.PHONY: build test lint clean install cross help

## build: Compile the binary for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## build-static: Compile a fully-static binary (Linux only)
build-static:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -o $(BINARY) .

## cross: Build binaries for all supported platforms
cross:
	@mkdir -p dist
	GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64  .
	GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64  .
	GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64   .
	GOOS=linux   GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64   .
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe .
	@echo "Cross-compiled binaries:"
	@ls -lh dist/

## test: Run all tests with coverage
test:
	go test -cover ./...

## test-race: Run tests with race detector
test-race:
	go test -race -cover ./...

## test-verbose: Run tests with verbose output
test-verbose:
	go test -v -cover ./...

## bench: Run benchmarks
bench:
	go test -bench=. -benchmem ./...

## lint: Run linters (requires golangci-lint)
lint:
	golangci-lint run ./...

## vet: Run go vet
vet:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## install: Install to GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

## run: Run directly with go run (interactive mode)
run:
	go run -ldflags "$(LDFLAGS)" . 

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
