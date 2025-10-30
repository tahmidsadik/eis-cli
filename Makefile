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
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) .
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
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 .
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64

.PHONY: build-darwin-amd64
build-darwin-amd64: ## Build for macOS amd64 (Intel) with version in filename
	@echo "Building $(BINARY_NAME) for macOS amd64..."
	@echo "Version: $(VERSION)"
	@if [ -z "$(OAUTH_CLIENT_ID)" ] || [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: OAuth credentials not set. Set BITBUCKET_OAUTH_CLIENT_ID and BITBUCKET_OAUTH_CLIENT_SECRET"; \
		exit 1; \
	fi
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64 .
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Build for macOS arm64 (Apple Silicon) with version in filename
	@echo "Building $(BINARY_NAME) for macOS arm64..."
	@echo "Version: $(VERSION)"
	@if [ -z "$(OAUTH_CLIENT_ID)" ] || [ -z "$(OAUTH_CLIENT_SECRET)" ]; then \
		echo "ERROR: OAuth credentials not set. Set BITBUCKET_OAUTH_CLIENT_ID and BITBUCKET_OAUTH_CLIENT_SECRET"; \
		exit 1; \
	fi
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64 .
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64

.PHONY: build-all-platforms
build-all-platforms: build-linux-amd64 build-darwin-amd64 build-darwin-arm64 ## Build for all supported platforms
	@echo "✓ All platform builds complete"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-*

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
