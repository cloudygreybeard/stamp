# stamp Makefile

BINARY := stamp
PREFIX ?= /usr/local
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/cloudygreybeard/stamp/cmd.Version=$(VERSION) \
	-X github.com/cloudygreybeard/stamp/cmd.Commit=$(COMMIT) \
	-X github.com/cloudygreybeard/stamp/cmd.Date=$(DATE)

.PHONY: all build test vet lint fmt cover check ci clean install snapshot deps help

## all: Build the binary (default target)
all: build

## build: Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: Run tests with race detector
test:
	go test -v -race -coverprofile=coverage.out ./...

## vet: Run go vet
vet:
	go vet ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## fmt: Format source code
fmt:
	gofmt -w .

## cover: Generate and open HTML coverage report
cover: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## check: Run vet and tests (no external tools required)
check: vet test

## ci: Run the full CI suite locally
ci: vet test lint

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -rf dist/

## install: Install to PREFIX/bin
install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

## snapshot: Build a snapshot release (no publish)
snapshot:
	goreleaser release --snapshot --clean

## deps: Download and tidy dependencies
deps:
	go mod download
	go mod tidy

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
