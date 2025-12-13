.PHONY: all build test clean run fmt vet lint build-all

# Binary name
BINARY_NAME=mc

# Version: use git tag if available, otherwise default
GIT_TAG=$(shell git describe --tags --abbrev=0 2>/dev/null)
VERSION?=$(if $(GIT_TAG),$(GIT_TAG),1.0.0)

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GORUN=$(GOCMD) run
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Output directory
DIST_DIR=dist

all: build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# Cross-platform builds (Unix only - uses SIGWINCH)
build-all: clean-dist build-linux build-darwin
	@echo "All builds completed in $(DIST_DIR)/"

build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64 .

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64 .

test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -v -cover ./...

clean: clean-dist
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

clean-dist:
	rm -rf $(DIST_DIR)

run:
	$(GORUN) .

fmt:
	$(GOFMT) ./...

vet:
	$(GOVET) ./...

tidy:
	$(GOMOD) tidy

deps:
	$(GOMOD) download

help:
	@echo "Available targets:"
	@echo "  build          - Build the binary for current OS"
	@echo "  build-all      - Build for all Unix platforms (Linux, macOS)"
	@echo "  build-linux    - Build for Linux (amd64, arm64)"
	@echo "  build-darwin   - Build for macOS (amd64, arm64)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Run the application"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  tidy           - Tidy go.mod"
	@echo "  deps           - Download dependencies"
	@echo "  help           - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION        - Set version (default: git tag or 1.0.0)"
	@echo "                   Example: make build-all VERSION=1.2.3"
	@echo ""
	@echo "Current version: $(VERSION)"