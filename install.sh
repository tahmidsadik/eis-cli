#!/usr/bin/env bash
set -euo pipefail

# EIS CLI Installer
# Downloads and installs the appropriate binary for the current system

REPO="cover42/eis-cli"
BASE_URL="https://bitbucket.org/${REPO}/downloads"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="eiscli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Error handling
error() {
    echo -e "${RED}Error:${NC} $1" >&2
    exit 1
}

info() {
    echo -e "${GREEN}ℹ${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Detect OS
detect_os() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux)
            echo "linux"
            ;;
        darwin)
            echo "darwin"
            ;;
        *)
            error "Unsupported operating system: $os"
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac
}

# Check for download command
check_download_cmd() {
    if command -v curl >/dev/null 2>&1; then
        echo "curl"
    elif command -v wget >/dev/null 2>&1; then
        echo "wget"
    else
        error "Neither curl nor wget is available. Please install one of them."
    fi
}

# Download binary
download_binary() {
    local os=$1
    local arch=$2
    local download_cmd=$3
    local filename="eiscli-${os}-${arch}-latest"
    local url="${BASE_URL}/${filename}"
    local temp_dir=$(mktemp -d)
    local temp_file="${temp_dir}/${filename}"
    
    info "Downloading ${filename}..."
    
    if [ "$download_cmd" = "curl" ]; then
        if ! curl -fL --progress-bar -o "$temp_file" "$url"; then
            rm -rf "$temp_dir"
            error "Failed to download binary from ${url}"
        fi
    else
        if ! wget --quiet --show-progress -O "$temp_file" "$url"; then
            rm -rf "$temp_dir"
            error "Failed to download binary from ${url}"
        fi
    fi
    
    echo "$temp_file"
}

# Install binary
install_binary() {
    local source_file=$1
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"
    
    # Check if source file exists
    if [ ! -f "$source_file" ]; then
        error "Downloaded file not found: $source_file"
    fi
    
    # Make binary executable
    chmod +x "$source_file"
    
    # Remove macOS quarantine flag if on macOS
    if [[ "$(uname -s)" == "Darwin" ]]; then
        info "Removing macOS quarantine flag..."
        xattr -d com.apple.quarantine "$source_file" 2>/dev/null || true
    fi
    
    # Check if installation directory exists and is writable
    if [ ! -d "$INSTALL_DIR" ]; then
        error "Installation directory does not exist: $INSTALL_DIR"
    fi
    
    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        info "Installation directory requires sudo access. You may be prompted for your password."
        sudo mv "$source_file" "$install_path"
        sudo chown root:wheel "$install_path" 2>/dev/null || true
    else
        mv "$source_file" "$install_path"
    fi
    
    # Verify installation
    if [ ! -f "$install_path" ]; then
        error "Installation failed. Binary not found at $install_path"
    fi
    
    # Verify binary is executable
    if [ ! -x "$install_path" ]; then
        warn "Binary is not executable. Attempting to fix..."
        chmod +x "$install_path" || error "Failed to make binary executable"
    fi
    
    info "Binary installed successfully to ${install_path}"
}

# Verify installation
verify_installation() {
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"
    
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local version=$($BINARY_NAME --version 2>/dev/null || echo "unknown")
        info "Installation verified successfully!"
        info "Version: $version"
        info "You can now use '${BINARY_NAME}' from anywhere."
        return 0
    else
        warn "Binary installed but not found in PATH."
        warn "Please ensure ${INSTALL_DIR} is in your PATH."
        warn "You can run the binary directly: ${install_path}"
        return 1
    fi
}

# Main installation flow
main() {
    info "EIS CLI Installer"
    info "=================="
    
    local os=$(detect_os)
    local arch=$(detect_arch)
    local download_cmd=$(check_download_cmd)
    
    info "Detected OS: ${os}"
    info "Detected architecture: ${arch}"
    info "Using ${download_cmd} for download"
    
    local temp_file=$(download_binary "$os" "$arch" "$download_cmd")
    
    # Cleanup function
    trap "rm -rf $(dirname "$temp_file")" EXIT
    
    install_binary "$temp_file"
    
    # Cleanup
    rm -rf "$(dirname "$temp_file")"
    trap - EXIT
    
    verify_installation
}

# Run main function
main "$@"

