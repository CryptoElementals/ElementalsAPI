# ElementalsAPI Makefile

# Go相关
GO = go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# 版本信息
TAG     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BLDTIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GOVER   ?= $(shell go version | awk '{print $$3}')

# 二进制输出目录
BIN_DIR = bin

# 二进制名称和主入口
APISERVER_BIN = ele-apiserver
APISERVER_MAIN = ./cmd/ele-apiserver

SCANNER_BIN = ele-scanner
SCANNER_MAIN = ./cmd/ele-scanner

ROOMSERVER_BIN = ele-roomserver
ROOMSERVER_MAIN = ./cmd/ele-roomserver

LOBBYSERVER_BIN = ele-lobbyserver
LOBBYSERVER_MAIN = ./cmd/ele-lobbyserver

BOTSERVER_BIN = ele-botserver
BOTSERVER_MAIN = ./cmd/ele-botserver

STAT_BIN = ele-stat
STAT_MAIN = ./cmd/ele-stat

TOOLS_BIN = ele-tools
TOOLS_MAIN = ./cmd/ele-tools

REDISSTREAM_BIN = ele-redis-stream
REDISSTREAM_MAIN = ./cmd/ele-redis-stream

LDFLAGS = -ldflags "-X 'main.TAG=$(TAG)' -X 'main.COMMIT=$(COMMIT)' -X 'main.BLDTIME=$(BLDTIME)' -X 'main.GOVER=$(GOVER)'"

.PHONY: all build apiserver scanner roomserver lobbyserver botserver stat tools redisstream clean deps lint check-persist help

all: build

build: apiserver scanner roomserver lobbyserver botserver stat tools redisstream

apiserver: $(BIN_DIR)
	@echo "Building $(APISERVER_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(APISERVER_BIN) $(APISERVER_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(APISERVER_BIN)"

scanner: $(BIN_DIR)
	@echo "Building $(SCANNER_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(SCANNER_BIN) $(SCANNER_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(SCANNER_BIN)"

roomserver: $(BIN_DIR)
	@echo "Building $(ROOMSERVER_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(ROOMSERVER_BIN) $(ROOMSERVER_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(ROOMSERVER_BIN)"

lobbyserver: $(BIN_DIR)
	@echo "Building $(LOBBYSERVER_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(LOBBYSERVER_BIN) $(LOBBYSERVER_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(LOBBYSERVER_BIN)"

botserver: $(BIN_DIR)
	@echo "Building $(BOTSERVER_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BOTSERVER_BIN) $(BOTSERVER_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(BOTSERVER_BIN)"

stat: $(BIN_DIR)
	@echo "Building $(STAT_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(STAT_BIN) $(STAT_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(STAT_BIN)"

tools: $(BIN_DIR)
	@echo "Building $(TOOLS_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(TOOLS_BIN) $(TOOLS_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(TOOLS_BIN)"

redisstream: $(BIN_DIR)
	@echo "Building $(REDISSTREAM_BIN)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(REDISSTREAM_BIN) $(REDISSTREAM_MAIN)
	@echo "Build completed: $(BIN_DIR)/$(REDISSTREAM_BIN)"

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	@echo "Cleaning build files..."
	rm -rf $(BIN_DIR)/$(APISERVER_BIN) $(BIN_DIR)/$(SCANNER_BIN) $(BIN_DIR)/$(ROOMSERVER_BIN) $(BIN_DIR)/$(LOBBYSERVER_BIN) $(BIN_DIR)/$(BOTSERVER_BIN) $(BIN_DIR)/$(STAT_BIN) $(BIN_DIR)/$(TOOLS_BIN) $(BIN_DIR)/$(REDISSTREAM_BIN)
	@echo "Clean completed"

deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping code linting"; \
	fi

check-persist:
	@./scripts/check-roomserver-persist-guard.sh

help:
	@echo "ElementalsAPI Makefile Usage:"
	@echo ""
	@echo "Build related:"
	@echo "  build         - Build all binaries (default)"
	@echo "  apiserver     - Build ele-apiserver only"
	@echo "  scanner       - Build ele-scanner only"
	@echo "  roomserver    - Build ele-roomserver only"
	@echo "  botserver     - Build ele-botserver only"
	@echo "  stat          - Build ele-stat only"
	@echo "  tools         - Build ele-tools (includes stress run subcommand)"
	@echo "  redisstream   - Build ele-redis-stream only (Redis Stream test tool)"
	@echo ""
	@echo "Other:"
	@echo "  clean         - Clean build files"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  check-persist - Fail if room_server prod code references SaveFullGameGraph"
	@echo "  help          - Show this help information"
	@echo ""
	@echo "Environment variables:"
	@echo "  TAG           - Git tag or version (default: git describe)"
	@echo "  COMMIT        - Git commit short hash (default: git rev-parse)"
	@echo "  BLDTIME       - Build time (default: now)"
	@echo "  GOVER         - Go version (default: go version)"
	@echo "  GOOS          - Target OS (default: current system)"
	@echo "  GOARCH        - Target architecture (default: current arch)" 