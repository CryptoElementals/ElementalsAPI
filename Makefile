# BeastRoyale Backend Makefile

# Variable definitions
APP_NAME = beast-royale-server
MAIN_FILE = main.go
CONFIG_FILE = config.yaml
BUILD_DIR = bin
LOG_DIR = bin/logs

# Go related variables
GO = go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GO_VERSION = $(shell go version | awk '{print $$3}')

# Version information
VERSION ?= 1.0.0
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Default target
.PHONY: all
all: clean build

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Create log directory
$(LOG_DIR):
	mkdir -p $(LOG_DIR)

# Build application
.PHONY: build
build: $(BUILD_DIR) $(LOG_DIR)
	@echo "Building $(APP_NAME)..."
	@echo "Go version: $(GO_VERSION)"
	@echo "Target platform: $(GOOS)/$(GOARCH)"
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build completed: $(BUILD_DIR)/$(APP_NAME)"

# Cross compile - Linux
.PHONY: build-linux
build-linux:
	@echo "Building Linux version..."
	GOOS=linux GOARCH=amd64 $(MAKE) build

# Cross compile - Windows
.PHONY: build-windows
build-windows:
	@echo "Building Windows version..."
	GOOS=windows GOARCH=amd64 $(MAKE) build

# Cross compile - macOS
.PHONY: build-darwin
build-darwin:
	@echo "Building macOS version..."
	GOOS=darwin GOARCH=amd64 $(MAKE) build

# Run application
.PHONY: run
run: build
	@echo "Starting $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME) -config $(CONFIG_FILE)

# Development mode run (no compilation)
.PHONY: dev
dev:
	@echo "Starting $(APP_NAME) in development mode..."
	$(GO) run $(MAIN_FILE) -config $(CONFIG_FILE)

# Test
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Test coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests and generating coverage report..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code formatting
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Code linting
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping code linting"; \
	fi

# Dependency management
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Update dependencies
.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

# Clean
.PHONY: clean
clean:
	@echo "Cleaning build files..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean completed"

# Install
.PHONY: install
install: build
	@echo "Installing $(APP_NAME)..."
	cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "Installation completed"

# Uninstall
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(APP_NAME)..."
	rm -f /usr/local/bin/$(APP_NAME)
	@echo "Uninstallation completed"

# Show help
.PHONY: help
help:
	@echo "BeastRoyale Backend Makefile Usage:"
	@echo ""
	@echo "Build related:"
	@echo "  build          - Build application"
	@echo "  build-linux    - Build Linux version"
	@echo "  build-windows  - Build Windows version"
	@echo "  build-darwin   - Build macOS version"
	@echo ""
	@echo "Run related:"
	@echo "  run            - Build and run application"
	@echo "  dev            - Run in development mode (no compilation)"
	@echo ""
	@echo "Test related:"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests and generate coverage report"
	@echo ""
	@echo "Code quality:"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo ""
	@echo "Dependency management:"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo ""
	@echo "Other:"
	@echo "  clean          - Clean build files"
	@echo "  install        - Install to system"
	@echo "  uninstall      - Uninstall from system"
	@echo "  help           - Show this help information"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION        - Application version (default: 1.0.0)"
	@echo "  GOOS           - Target OS (default: current system)"
	@echo "  GOARCH         - Target architecture (default: current architecture)"

# Show version information
.PHONY: version
version:
	@echo "BeastRoyale Backend Server"
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Go version: $(GO_VERSION)"
	@echo "Target platform: $(GOOS)/$(GOARCH)"

# Create release package
.PHONY: release
release: clean build
	@echo "Creating release package..."
	@mkdir -p release
	@cp $(BUILD_DIR)/$(APP_NAME) release/
	@cp $(CONFIG_FILE) release/
	@cp README.md release/ 2>/dev/null || echo "README.md not found, skipping"
	@tar -czf release/$(APP_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz -C release .
	@rm -rf release/$(APP_NAME) release/$(CONFIG_FILE) release/README.md
	@echo "Release package created: release/$(APP_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz" 