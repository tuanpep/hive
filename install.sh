#!/bin/bash
set -e

# Configuration
REPO_OWNER="tuanpep"
REPO_NAME="hive"
BINARY_NAME="hive"
INSTALL_DIR="/usr/local/bin"

# Detect OS and Arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" == "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" == "aarch64" ] || [ "$ARCH" == "arm64" ]; then
    ARCH="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

echo "Detected Platform: $OS/$ARCH"

# Determine latest release or use specified version
if [ -z "$1" ]; then
    echo "Fetching latest version..."
    LATEST_RELEASE_URL="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
    TAG_NAME=$(curl -sL $LATEST_RELEASE_URL | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$TAG_NAME" ]; then
        echo "Error: Could not determine latest release. Have releases been published?"
        exit 1
    fi
else
    TAG_NAME=$1
fi

echo "Installing HIVE $TAG_NAME..."

# Construct Download URL
# Pattern matches GoReleaser naming: hive_linux_amd64.tar.gz / hive_darwin_arm64.tar.gz
ASSET_NAME="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$TAG_NAME/$ASSET_NAME"

echo "Downloading from $DOWNLOAD_URL..."
TMP_DIR=$(mktemp -d)
curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/hive.tar.gz"

echo "Extracting..."
tar -xzf "$TMP_DIR/hive.tar.gz" -C "$TMP_DIR"

echo "Installing to $INSTALL_DIR (requires sudo)..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
    sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Cleanup
rm -rf "$TMP_DIR"

echo "✅ HIVE installed successfully!"
echo "Run 'hive' to start the swarm."
