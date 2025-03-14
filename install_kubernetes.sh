#!/bin/bash

# Kubernetes Installation Script
# Supports macOS, Linux, and Windows (via WSL)

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect Operating System
OS=$(uname -s)
ARCH=$(uname -m)

# Kubernetes and Tools Versions
KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
MINIKUBE_VERSION="v1.32.0"
HELM_VERSION="v3.14.0"

# Logging function
log() {
    echo -e "${GREEN}[K8S INSTALLER]${NC} $1"
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

# Install Docker (prerequisite)
install_docker() {
    log "Installing Docker..."
    case "$OS" in
        Darwin)
            # macOS Docker Desktop installation
            brew install --cask docker
            ;;
        Linux)
            # Linux Docker installation
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                case "$ID" in
                    ubuntu|debian)
                        sudo apt-get update
                        sudo apt-get install -y docker.io
                        ;;
                    fedora|rhel|centos)
                        sudo dnf install -y docker
                        ;;
                    *)
                        error "Unsupported Linux distribution"
                        ;;
                esac
            fi
            ;;
        *)
            error "Docker installation not supported on this OS"
            ;;
    esac

    # Start and enable Docker service
    sudo systemctl start docker
    sudo systemctl enable docker
}

# Install kubectl
install_kubectl() {
    log "Installing kubectl ${KUBECTL_VERSION}"
    
    curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${OS,,}/${ARCH}/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
}

# Install Minikube
install_minikube() {
    log "Installing Minikube ${MINIKUBE_VERSION}"
    
    curl -LO "https://storage.googleapis.com/minikube/releases/${MINIKUBE_VERSION}/minikube-${OS,,}-${ARCH}"
    chmod +x "minikube-${OS,,}-${ARCH}"
    sudo mv "minikube-${OS,,}-${ARCH}" /usr/local/bin/minikube
}

# Install Helm
install_helm() {
    log "Installing Helm ${HELM_VERSION}"
    
    curl -fsSL -o helm.tar.gz "https://get.helm.sh/helm-${HELM_VERSION}-${OS,,}-${ARCH}.tar.gz"
    tar -zxvf helm.tar.gz
    sudo mv "${OS,,}-${ARCH}/helm" /usr/local/bin/
    rm -rf helm.tar.gz "${OS,,}-${ARCH}"
}

# Configure Kubernetes cluster
configure_cluster() {
    log "Starting Minikube cluster"
    minikube start --driver=docker

    log "Setting Minikube as default context"
    kubectl config use-context minikube
}

# Verify installations
verify_installations() {
    log "Verifying Kubernetes installations"
    
    kubectl version --client
    minikube version
    helm version
}

# Main installation process
main() {
    log "Starting Kubernetes Development Environment Setup"

    # Validate and adjust architecture
    validate_arch

    # Require sudo for system-wide installation
    if [[ "$EUID" -ne 0 ]]; then
        error "Please run as root or with sudo"
    fi

    # Install prerequisites and tools
    install_docker
    install_kubectl
    install_minikube
    install_helm

    # Configure cluster
    configure_cluster

    # Verify installations
    verify_installations

    log "Kubernetes development environment successfully installed!"
    log "To interact with your cluster, use 'kubectl' and 'minikube' commands"
}

# Execute main function
main