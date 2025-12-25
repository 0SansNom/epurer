#!/bin/bash
set -e

# Mac Developer Cleaner Installation Script

REPO="0SansNom/mac-dev-clean"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="mac-dev-clean"

echo "🧹 Mac Developer Cleaner - Installation Script"
echo ""

# Check if running on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ Error: This tool is designed for macOS only"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH_NAME="amd64"
elif [[ "$ARCH" == "arm64" ]]; then
    ARCH_NAME="arm64"
else
    echo "❌ Error: Unsupported architecture: $ARCH"
    exit 1
fi

echo "📦 Detected: macOS $ARCH_NAME"

# Check if we're in the project directory
if [[ -f "bin/$BINARY_NAME" ]]; then
    echo "ℹ️  Installing from local build..."
    BINARY_PATH="bin/$BINARY_NAME"
else
    echo "ℹ️  Downloading latest release..."

    # Get latest release version
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [[ -z "$LATEST_VERSION" ]]; then
        echo "❌ Error: Could not fetch latest version"
        exit 1
    fi

    echo "📥 Downloading version $LATEST_VERSION..."

    # Download URL
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}_${LATEST_VERSION#v}_darwin_${ARCH_NAME}.tar.gz"

    # Download and extract
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    if ! curl -sL "$DOWNLOAD_URL" -o "$BINARY_NAME.tar.gz"; then
        echo "❌ Error: Failed to download release"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    tar -xzf "$BINARY_NAME.tar.gz"
    BINARY_PATH="$TMP_DIR/$BINARY_NAME"
fi

# Install binary
echo "📦 Installing to $INSTALL_DIR..."

if [[ -w "$INSTALL_DIR" ]]; then
    cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    echo "🔐 Requesting sudo privileges for installation..."
    sudo cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

# Cleanup if we downloaded
if [[ -d "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "Usage:"
echo "  $BINARY_NAME detect      # Detect installed tools"
echo "  $BINARY_NAME report      # Generate cleanup report"
echo "  $BINARY_NAME clean       # Clean development caches"
echo "  $BINARY_NAME smart       # Smart automatic cleanup"
echo ""
echo "For more information, run: $BINARY_NAME --help"
