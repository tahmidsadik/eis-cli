# EIS CLI Makefile
# Build with OAuth credentials injected at build time

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# OAuth credentials from environment (for build-time injection)
# These should be set as repository variables in Bitbucket Pipelines
OAUTH_CLIENT_ID ?= $(BITBUCKET_OAUTH_CLIENT_ID)
OAUTH_CLIENT_SECRET ?= $(BITBUCKET_OAUTH_CLIENT_SECRET)

# Build output
BINARY_NAME = eiscli
BUILD_DIR = .

# Platform-specific builds
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Go build flags
LDFLAGS = -X 'bitbucket.org/cover42/eiscli/internal/bitbucket.DefaultClientID=$(OAUTH_CLIENT_ID)' \
          -X 'bitbucket.org/cover42/eiscli/internal/bitbucket.DefaultClientSecret=$(OAUTH_CLIENT_SECRET)'

# Optimized build flags for minimal binary size
# -s: omit symbol table and debug information
# -w: omit DWARF symbol table
# -trimpath: remove file system paths from compiled binary
# -buildmode=exe: build as executable (default, but explicit for clarity)
BUILD_FLAGS = -ldflags "$(LDFLAGS) -s -w" -trimpath -buildmode=exe
BUILD_ENV = CGO_ENABLED=0

.PHONY: build
build: ## Build the CLI (without OAuth credentials - for development)
	@echo "Building $(BINARY_NAME)..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(GIT_COMMIT)"
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-with-oauth
build-with-oauth: ## Build with OAuth credentials injected (requires BITBUCKET_OAUTH_CLIENT_ID and BITBUCKET_OAUTH_CLIENT_SECRET env vars)
	@echo "Building $(BINARY_NAME) with OAuth credentials..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(GIT_COMMIT)"
	@if [ -z "$(OAUTH_CLIENT_ID)" ]; then \
		echo "ERROR: BITBUCKET_OAUTH_CLIENT_ID environment variable not set"; \
		echo "Usage: BITBUCKET_OAUTH_CLIENT_ID=xxx BITBUCKET_OAUTH_CLIENT_SECRET=yyy make build-with-oauth"; \
		exit 1; \
	fi
	@if [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: BITBUCKET_OAUTH_CLIENT_SECRET environment variable not set"; \
		exit 1; \
	fi
	@echo "OAuth Client ID: $(OAUTH_CLIENT_ID)"
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✓ Build complete with OAuth credentials: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-release
build-release: ## Build optimized release binary with OAuth credentials
	@echo "Building release binary..."
	@echo "Version: $(VERSION)"
	@if [ -z "$(OAUTH_CLIENT_ID)" ] || [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: OAuth credentials not set. Set BITBUCKET_OAUTH_CLIENT_ID and BITBUCKET_OAUTH_CLIENT_SECRET"; \
		exit 1; \
	fi
	$(BUILD_ENV) go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✓ Release build complete: $(BUILD_DIR)/$(BINARY_NAME)"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: build-linux-amd64
build-linux-amd64: ## Build for Linux amd64 with version in filename
	@echo "Building $(BINARY_NAME) for Linux amd64..."
	@echo "Version: $(VERSION)"
	@if [ -z "$(OAUTH_CLIENT_ID)" ] || [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: OAuth credentials not set. Set BITBUCKET_OAUTH_CLIENT_ID and BITBUCKET_OAUTH_CLIENT_SECRET"; \
		exit 1; \
	fi
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-$(VERSION) .
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-$(VERSION)"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-$(VERSION)

.PHONY: build-darwin-amd64
build-darwin-amd64: ## Build for macOS amd64 (Intel) with version in filename
	@echo "Building $(BINARY_NAME) for macOS amd64..."
	@echo "Version: $(VERSION)"
	@if [ -z "$(OAUTH_CLIENT_ID)" ] || [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: OAuth credentials not set. Set BITBUCKET_OAUTH_CLIENT_ID and BITBUCKET_OAUTH_CLIENT_SECRET"; \
		exit 1; \
	fi
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64-$(VERSION) .
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64-$(VERSION)"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64-$(VERSION)

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Build for macOS arm64 (Apple Silicon) with version in filename
	@echo "Building $(BINARY_NAME) for macOS arm64..."
	@echo "Version: $(VERSION)"
	@if [ -z "$(OAUTH_CLIENT_ID)" ] || [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: OAuth credentials not set. Set BITBUCKET_OAUTH_CLIENT_ID and BITBUCKET_OAUTH_CLIENT_SECRET"; \
		exit 1; \
	fi
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64-$(VERSION) .
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64-$(VERSION)"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64-$(VERSION)

.PHONY: build-all-platforms
build-all-platforms: build-linux-amd64 build-darwin-amd64 build-darwin-arm64 ## Build for all supported platforms
	@echo "✓ All platform builds complete"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-*-$(VERSION)

.PHONY: create-latest-binaries
create-latest-binaries: ## Create latest binaries by copying versioned binaries
	@echo "Creating latest binaries from versioned builds..."
	@if [ -f "$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-$(VERSION)" ]; then \
		cp "$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-$(VERSION)" "$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-latest"; \
		echo "✓ Created $(BINARY_NAME)-linux-amd64-latest"; \
	fi
	@if [ -f "$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64-$(VERSION)" ]; then \
		cp "$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64-$(VERSION)" "$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64-latest"; \
		echo "✓ Created $(BINARY_NAME)-darwin-amd64-latest"; \
	fi
	@if [ -f "$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64-$(VERSION)" ]; then \
		cp "$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64-$(VERSION)" "$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64-latest"; \
		echo "✓ Created $(BINARY_NAME)-darwin-arm64-latest"; \
	fi
	@echo "✓ Latest binaries created"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-*-latest 2>/dev/null || true

.PHONY: build-all-platforms-compressed
build-all-platforms-compressed: ## Build for all platforms and compress with UPX (requires UPX installed)
	@echo "Building and compressing binaries for all platforms..."
	@if ! command -v upx >/dev/null 2>&1; then \
		echo "ERROR: UPX not installed. Install from https://upx.github.io/"; \
		echo "   macOS: brew install upx"; \
		echo "   Linux: apt-get install upx-ucl || yum install upx"; \
		exit 1; \
	fi
	@$(MAKE) build-all-platforms
	@echo "Compressing binaries with UPX..."
	@for binary in $(BUILD_DIR)/$(BINARY_NAME)-*-$(VERSION); do \
		if [ -f "$$binary" ]; then \
			echo "Compressing $$binary..."; \
			upx --best --lzma "$$binary" || echo "Warning: UPX compression failed for $$binary"; \
		fi \
	done
	@echo "✓ All platform builds complete (compressed)"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-*-$(VERSION)

.PHONY: compress-binary
compress-binary: ## Compress existing binary with UPX (requires UPX installed)
	@if [ ! -f "$(BUILD_DIR)/$(BINARY_NAME)" ]; then \
		echo "ERROR: Binary $(BUILD_DIR)/$(BINARY_NAME) not found. Build it first."; \
		exit 1; \
	fi
	@if ! command -v upx >/dev/null 2>&1; then \
		echo "ERROR: UPX not installed. Install from https://upx.github.io/"; \
		exit 1; \
	fi
	@echo "Compressing $(BUILD_DIR)/$(BINARY_NAME) with UPX..."
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)
	@upx --best --lzma $(BUILD_DIR)/$(BINARY_NAME)
	@echo "✓ Binary compressed"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: analyze-binary-size
analyze-binary-size: build-release ## Analyze binary size and dependencies
	@echo "Analyzing binary size..."
	@echo "Binary: $(BUILD_DIR)/$(BINARY_NAME)"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)
	@echo ""
	@echo "Checking dependencies..."
	@go list -m all | wc -l | xargs echo "Total dependencies:"
	@echo ""
	@echo "Top 10 largest dependencies (by binary size contribution):"
	@go list -m -f '{{.Path}} {{.Version}}' all | head -10

.PHONY: install
install: build ## Install the CLI to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install .
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

.PHONY: install-with-oauth
install-with-oauth: ## Install with OAuth credentials injected
	@echo "Installing $(BINARY_NAME) with OAuth credentials..."
	@if [ -z "$(OAUTH_CLIENT_ID)" ] || [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: OAuth credentials not set"; \
		exit 1; \
	fi
	go install -ldflags "$(LDFLAGS)" .
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/"; \
	fi

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

.PHONY: verify-oauth-build
verify-oauth-build: build-with-oauth ## Build with OAuth and verify credentials are injected
	@echo ""
	@echo "Verifying OAuth credentials in binary..."
	@if strings $(BUILD_DIR)/$(BINARY_NAME) | grep -q "$(OAUTH_CLIENT_ID)"; then \
		echo "✓ OAuth Client ID found in binary"; \
	else \
		echo "✗ OAuth Client ID NOT found in binary"; \
		exit 1; \
	fi
	@echo "✓ OAuth credentials successfully injected"

.PHONY: dist
dist: build-release ## Create distribution package
	@echo "Creating distribution package..."
	@mkdir -p dist
	@cp $(BUILD_DIR)/$(BINARY_NAME) dist/
	@cp README.md dist/ 2>/dev/null || true
	@cp config.yaml.example dist/ 2>/dev/null || true
	@tar -czf dist/$(BINARY_NAME)-$(VERSION).tar.gz -C dist $(BINARY_NAME) README.md config.yaml.example 2>/dev/null || \
		tar -czf dist/$(BINARY_NAME)-$(VERSION).tar.gz -C dist $(BINARY_NAME)
	@echo "✓ Distribution package: dist/$(BINARY_NAME)-$(VERSION).tar.gz"
	@ls -lh dist/$(BINARY_NAME)-$(VERSION).tar.gz

.DEFAULT_GOAL := help
