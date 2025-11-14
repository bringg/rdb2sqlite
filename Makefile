# Makefile for rdb2sqlite

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build variables
BINARY_NAME := rdb2sqlite
BUILD_DIR := ./cmd/rdb2sqlite
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

.PHONY: all build clean test run help install deps

## all: Build the binary
all: build

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(BUILD_DIR)

## clean: Remove build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

## test: Run tests
test:
	$(GOTEST) -v ./...

## run: Build and run with example parameters (requires dump.rdb)
run: build
	./$(BINARY_NAME) --rdb dump.rdb --db test.db

## install: Install the binary to GOPATH/bin
install:
	$(GOCMD) install $(LDFLAGS) $(BUILD_DIR)

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## fmt: Format code
fmt:
	$(GOCMD) fmt ./...

## vet: Run go vet
vet:
	$(GOCMD) vet ./...

## lint: Run golangci-lint (if installed)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed"; exit 1)
	golangci-lint run

## version: Show version information
version:
	@echo "Version:    $(VERSION)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Git Commit: $(GIT_COMMIT)"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
