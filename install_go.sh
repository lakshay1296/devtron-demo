#!/bin/bash

# Go Language Installation Script
# Supports macOS, Linux, and Windows (via WSL)

set -e

# Detect Operating System
OS=$(uname -s)
ARCH=$(uname -m)

# Go Version (latest stable as of script creation)
GO_VERSION="1.21.5"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[GO INSTALLER]${NC} $1"
}

# Error handling function
error() {
    echo -e "${YELLOW}[ERROR]${NC} $1"
    exit 1
}

# Validate architecture
validate_arch() {
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        arm64) ARCH="arm64" ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac
}

# Download Go
download_go() {
    local download_url=""
    
    case "$OS" in
        Darwin)
            download_url="https://golang.org/dl/go${GO_VERSION}.darwin-${ARCH}.tar.gz"
            ;;
        Linux)
            download_url="https://golang.org/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac

    log "Downloading Go ${GO_VERSION} for ${OS} ${ARCH}"
    curl -L "$download_url" -o go.tar.gz
}

# Install Go
install_go() {
    # Remove existing Go installation if exists
    sudo rm -rf /usr/local/go

    # Extract Go
    sudo tar -C /usr/local -xzf go.tar.gz

    # Clean up download
    rm go.tar.gz
}

# Configure environment
configure_environment() {
    log "Configuring Go environment"

    # Detect shell
    if [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    else
        SHELL_CONFIG="$HOME/.bashrc"
    fi

    # Add Go paths
    {
        echo ""
        echo "# Go Language Configuration"
        echo 'export GOROOT=/usr/local/go'
        echo 'export GOPATH=$HOME/go'
        echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin'
    } >> "$SHELL_CONFIG"

    # Create Go workspace directories
    mkdir -p "$HOME/go/src"
    mkdir -p "$HOME/go/pkg"
    mkdir -p "$HOME/go/bin"
}

# Verify installation
verify_installation() {
    log "Verifying Go installation"
    go version
    go env
}

# Main installation process
main() {
    log "Starting Go Language Installation"

    # Validate and adjust architecture
    validate_arch

    # Require sudo for system-wide installation
    if [[ "$EUID" -ne 0 ]]; then
        error "Please run as root or with sudo"
    fi

    # Download and install
    download_go
    install_go
    configure_environment
    verify_installation

    log "Go ${GO_VERSION} successfully installed!"
    log "Please restart your terminal or run 'source ${SHELL_CONFIG}'"
}

# Execute main function
main