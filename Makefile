# Variables
BINARY_NAME=terraform-mcp-server
VERSION?=dev
GO=go
DOCKER=docker

# Build flags
LDFLAGS=-ldflags="-s -w -X main.version=${VERSION} -X main.commit=$(shell git rev-parse HEAD) -X main.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: all build test clean docker-build help

# Default target
all: build

# Build the binary
build:
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) cmd/terraform-mcp-server/main.go

# Run tests
test:
	$(GO) test -v ./...

# Run e2e tests
test-e2e:
	$(GO) test -v --tags e2e ./e2e

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	$(GO) clean

# Download dependencies
deps:
	$(GO) mod download

# Build docker image
docker-build:
	$(DOCKER) build --build-arg VERSION=$(VERSION) -t $(BINARY_NAME):$(VERSION) .

# Run docker container
# docker-run:
# 	$(DOCKER) run -it --rm $(BINARY_NAME):$(VERSION)

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Build the binary (default)"
	@echo "  build        - Build the binary"
	@echo "  test         - Run all tests"
	@echo "  test-e2e     - Run end-to-end tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  docker-build - Build docker image"
	@echo "  help         - Show this help message"
