#!/bin/bash

# Script to update mcp-kubernetes to the latest version
# Usage: ./update-mcp-k8s.sh [version]
# If no version specified, downloads the latest

set -e

REPO="Azure/mcp-kubernetes"
BINARY_NAME="mcp-kubernetes"
INSTALL_PATH="/usr/local/bin/mcp-kubernetes"

# Detect architecture
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Map architecture names
case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Show help
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [version]"
    echo "  version: Specific version to install (e.g., v0.0.9)"
    echo "  If no version specified, installs the latest version"
    echo ""
    echo "Examples:"
    echo "  $0           # Install latest version"
    echo "  $0 v0.0.9    # Install specific version"
    exit 0
fi

# Get version (latest if not specified)
if [ -z "$1" ]; then
    echo "Fetching latest version..."
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
else
    VERSION="$1"
fi

echo "Updating mcp-kubernetes to version: $VERSION"

# Construct download URL
BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_FILE"

# Create temporary file
TEMP_FILE=$(mktemp)

echo "Downloading from: $DOWNLOAD_URL"
curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL"

# Make executable
chmod +x "$TEMP_FILE"

# Test the binary
echo "Testing downloaded binary..."
"$TEMP_FILE" --version

# Install (requires sudo)
echo "Installing to $INSTALL_PATH (requires sudo)..."
sudo mv "$TEMP_FILE" "$INSTALL_PATH"

echo "✅ Successfully updated mcp-kubernetes to $VERSION"
echo "Current version:"
mcp-kubernetes --version
