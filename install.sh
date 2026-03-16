#!/usr/bin/env bash
# aicoder installation script
# Usage: curl -fsSL https://raw.githubusercontent.com/iminders/aicoder/main/install.sh | bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="iminders/aicoder"
BINARY_NAME="aicoder"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
detect_platform() {
    local os
    local arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)
            echo -e "${RED}Error: Unsupported operating system$(uname -s)${NC}"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="x86_64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)
            echo -e "${RED}Error: Unsupported architecture $(uname -m)${NC}"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        echo -e "${RED}Error: Failed to fetch latest version${NC}"
        exit 1
    fi

    echo "$version"
}

# Download and install binary
install_binary() {
    local platform="$1"
    local version="$2"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${version#v}_${platform}.tar.gz"

    if [[ "$platform" == *"windows"* ]]; then
        download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${version#v}_${platform}.zip"
    fi

    echo -e "${BLUE}Downloading ${BINARY_NAME} ${version} for ${platform}...${NC}"

    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf ${tmp_dir}" EXIT

    if ! curl -fsSL "$download_url" -o "${tmp_dir}/archive"; then
        echo -e "${RED}Error: Failed to download from ${download_url}${NC}"
        exit 1
    fi

    echo -e "${BLUE}Extracting archive...${NC}"
    cd "$tmp_dir"

    if [[ "$download_url" == *.tar.gz ]]; then
        tar -xzf archive
    elif [[ "$download_url" == *.zip ]]; then
        unzip -q archive
    fi

    # Find the binary
    local binary_path
    if [[ "$platform" == *"windows"* ]]; then
        binary_path="${BINARY_NAME}.exe"
    else
        binary_path="${BINARY_NAME}"
    fi

    if [ ! -f "$binary_path" ]; then
        echo -e "${RED}Error: Binary not found in archive${NC}"
        exit 1
    fi

    # Install binary
    echo -e "${BLUE}Installing to ${INSTALL_DIR}...${NC}"

    if [ ! -d "$INSTALL_DIR" ]; then
        echo -e "${YELLOW}Creating ${INSTALL_DIR}...${NC}"
        sudo mkdir -p "$INSTALL_DIR"
    fi

    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    echo -e "${GREEN}✓ ${BINARY_NAME} ${version} installed successfully!${NC}"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        local installed_version
        installed_version=$("$BINARY_NAME" --version 2>&1 | head -n1)
        echo -e "${GREEN}✓ Verification successful: ${installed_version}${NC}"
        return 0
    else
        echo -e "${YELLOW}Warning: ${BINARY_NAME} not found in PATH${NC}"
        echo -e "${YELLOW}You may need to add ${INSTALL_DIR} to your PATH${NC}"
        echo -e "${YELLOW}Run: export PATH=\"${INSTALL_DIR}:\$PATH\"${NC}"
        return 1
    fi
}

# Print usage instructions
print_usage() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  aicoder installation complete!${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Quick start:"
    echo "  1. Set your API key:"
    echo "     export ANTHROPIC_API_KEY=\"sk-ant-...\""
    echo ""
    echo "  2. Run aicoder:"
    echo "     aicoder"
    echo ""
    echo "  3. Get help:"
    echo "     aicoder --help"
    echo ""
    echo "Documentation: https://github.com/${REPO}"
    echo ""
}

# Main installation flow
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  aicoder installer${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # Check dependencies
    for cmd in curl tar; do
        if ! command -v "$cmd" &> /dev/null; then
            echo -e "${RED}Error: Required command '$cmd' not found${NC}"
            exit 1
        fi
    done

    local platform
    platform=$(detect_platform)
    echo -e "${BLUE}Detected platform: ${platform}${NC}"

    local version
    version=$(get_latest_version)
    echo -e "${BLUE}Latest version: ${version}${NC}"
    echo ""

    install_binary "$platform" "$version"
    echo ""

    verify_installation
    print_usage
}

# Run main function
main "$@"
