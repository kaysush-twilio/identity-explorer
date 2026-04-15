.PHONY: build run test lint clean release-snapshot

# Build variables
BINARY_NAME=identity-explorer
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/identity-explorer

# Run the application
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -rf dist/

# Create a snapshot release (for testing)
release-snapshot:
	goreleaser release --snapshot --clean

# Install locally
install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  run             - Build and run the application"
	@echo "  test            - Run tests with coverage"
	@echo "  lint            - Run golangci-lint"
	@echo "  clean           - Remove build artifacts"
	@echo "  release-snapshot- Create a snapshot release"
	@echo "  install         - Install binary to GOPATH/bin"
	@echo "  deps            - Download and tidy dependencies"
