#!/bin/bash

# DevBase Installation Script for Linux/macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/maleesha-pramud/devbase/main/install.sh | bash

set -e

REPO="maleesha-pramud/devbase"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="devbase"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        *)
            echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv6l)
            ARCH="arm"
            ;;
        *)
            echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    echo -e "${GREEN}Detected platform: ${OS}-${ARCH}${NC}"
}

# Get latest release version
get_latest_version() {
    echo -e "${YELLOW}Fetching latest version...${NC}"
    LATEST_VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$LATEST_VERSION" ]; then
        echo -e "${RED}Error: Could not fetch latest version${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Latest version: ${LATEST_VERSION}${NC}"
}

# Download and install binary
install_binary() {
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/${LATEST_VERSION}/devbase-${OS}-${ARCH}.tar.gz"
    TEMP_DIR=$(mktemp -d)
    
    echo -e "${YELLOW}Downloading DevBase from ${DOWNLOAD_URL}...${NC}"
    
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_DIR/devbase.tar.gz"; then
        echo -e "${RED}Error: Failed to download binary${NC}"
        echo -e "${YELLOW}URL: ${DOWNLOAD_URL}${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    echo -e "${YELLOW}Extracting archive...${NC}"
    tar -xzf "$TEMP_DIR/devbase.tar.gz" -C "$TEMP_DIR"
    
    echo -e "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TEMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        echo -e "${YELLOW}Need sudo privileges to install to ${INSTALL_DIR}${NC}"
        sudo mv "$TEMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    rm -rf "$TEMP_DIR"
    
    echo -e "${GREEN}✓ DevBase installed successfully!${NC}"
}

# Verify installation
verify_installation() {
    if command -v devbase &> /dev/null; then
        VERSION=$(devbase --version 2>/dev/null || echo "unknown")
        echo -e "${GREEN}✓ Installation verified: ${VERSION}${NC}"
        echo ""
        echo -e "${GREEN}Run 'devbase' to get started!${NC}"
    else
        echo -e "${YELLOW}Warning: 'devbase' command not found in PATH${NC}"
        echo -e "${YELLOW}You may need to add ${INSTALL_DIR} to your PATH${NC}"
        echo -e "${YELLOW}Or restart your terminal session${NC}"
    fi
}

# Main installation process
main() {
    echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   DevBase Installation Script          ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
    echo ""
    
    detect_platform
    get_latest_version
    install_binary
    verify_installation
    
    echo ""
    echo -e "${GREEN}Installation complete! 🎉${NC}"
    echo ""
    echo -e "To uninstall, run:"
    echo -e "  ${YELLOW}sudo rm ${INSTALL_DIR}/${BINARY_NAME}${NC}"
    echo -e "  ${YELLOW}rm ~/devbase.db${NC} (optional - removes database)"
}

main
