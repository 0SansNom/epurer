#!/bin/bash
set -e

# Épurer Installation Script

REPO="0SansNom/epurer"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="epurer"

echo "🧹 Épurer - Installation Script"
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
    VERSION_NUM="${LATEST_VERSION#v}"
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NUM}_darwin_${ARCH_NAME}.tar.gz"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${ARCHIVE_NAME}"
    CHECKSUMS_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}_${VERSION_NUM}_checksums.txt"

    # Download and extract
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    if ! curl -sL "$DOWNLOAD_URL" -o "$ARCHIVE_NAME"; then
        echo "❌ Error: Failed to download release"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    if ! curl -sL "$CHECKSUMS_URL" -o "checksums.txt"; then
        echo "❌ Error: Failed to download checksums"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    if ! grep -E "[[:space:]]${ARCHIVE_NAME}\$" checksums.txt | shasum -a 256 -c -; then
        echo "❌ Error: Checksum verification failed"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    tar -xzf "$ARCHIVE_NAME"
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
