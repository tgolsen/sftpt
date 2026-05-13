# sftpt Makefile
# A simple command-line SFTP utility for Mac

# Build variables
BINARY_NAME=sftpt
CMD_DIR=./cmd/sftpt
BUILD_DIR=./build
DIST_DIR=./dist

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Version and build info
VERSION?=0.1.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

.PHONY: all build clean test lint format install help deps security coverage

# Default target
all: deps format lint test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for distribution (optimized)
build-dist:
	@echo "Building $(BINARY_NAME) for distribution..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(DIST_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Distribution build completed: $(DIST_DIR)/$(BINARY_NAME)"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f $(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover ./...
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
format:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Lint code
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Security scan
security:
	@echo "Running security scan..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

# Install binary locally
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation completed"

# Uninstall binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall completed"

# Development build (faster, less optimized)
dev: format
	@echo "Building for development..."
	$(GOBUILD) -o $(BINARY_NAME) $(CMD_DIR)

# Run the application (for testing)
run: dev
	./$(BINARY_NAME) --help

# Cross-platform builds
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)

	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)

	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

	@echo "Cross-platform builds completed in $(DIST_DIR)/"

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && \
	for binary in $(BINARY_NAME)-*; do \
		tar -czf $$binary.tar.gz $$binary; \
		echo "Created: $$binary.tar.gz"; \
	done

# Quality gate - run all checks
quality: deps format lint security test
	@echo "All quality checks passed!"

# Development workflow - quick checks before commit
pre-commit: format lint test
	@echo "Pre-commit checks completed!"

# Help target
help:
	@echo "Available targets:"
	@echo "  all         - Run deps, format, lint, test, and build"
	@echo "  build       - Build the application"
	@echo "  build-dist  - Build optimized for distribution"
	@echo "  build-all   - Cross-platform builds"
	@echo "  release     - Create release archives"
	@echo "  deps        - Install dependencies"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  coverage    - Run tests with coverage report"
	@echo "  format      - Format code"
	@echo "  lint        - Lint code"
	@echo "  security    - Run security scan"
	@echo "  install     - Install binary to /usr/local/bin"
	@echo "  uninstall   - Remove binary from /usr/local/bin"
	@echo "  dev         - Quick development build"
	@echo "  run         - Build and run application"
	@echo "  quality     - Run all quality checks"
	@echo "  pre-commit  - Run pre-commit checks"
	@echo "  help        - Show this help"

# Development targets for testing SFTP functionality
# These require a test SFTP server setup

test-sftp-setup:
	@echo "Setting up test SFTP server..."
	@echo "Note: This requires Docker to be installed"
	@echo "TODO: Implement test SFTP server setup"

test-integration: build
	@echo "Running integration tests..."
	@echo "Note: This requires test SFTP server to be running"
	@echo "TODO: Implement integration tests"
	$(GOTEST) -tags=integration ./test/integration/...