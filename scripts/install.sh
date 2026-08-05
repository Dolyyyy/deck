#!/usr/bin/env bash
set -e

# Deck automated 1-line installer for Linux and macOS
REPO="Dolyyyy/deck"
BINARY_NAME="deck"

echo "⚡ Installing deck..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7*) ARCH="arm" ;;
    *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "Unsupported OS: $OS" && exit 1 ;;
esac

LATEST_RELEASE=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "v0.1.0")
TARBALL="deck_${LATEST_RELEASE#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST_RELEASE}/${TARBALL}"

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "⬇️  Downloading ${TARBALL} from ${URL}..."
if curl -fsSL "$URL" -o "${TEMP_DIR}/${TARBALL}" 2>/dev/null; then
    tar -xzf "${TEMP_DIR}/${TARBALL}" -C "$TEMP_DIR"
    BIN_SRC="${TEMP_DIR}/${BINARY_NAME}"
else
    echo "⚠️  Pre-built release not found, building from source via go install..."
    if command -v go >/dev/null 2>&1; then
        go install "github.com/${REPO}/cmd/deck@latest"
        echo "✓ Successfully installed via go install!"
        exit 0
    else
        echo "❌ Pre-built binary not found and Go is not installed."
        exit 1
    fi
fi

INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

mv "$BIN_SRC" "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo "🎉 Successfully installed deck to ${INSTALL_DIR}/${BINARY_NAME}!"
echo "👉 Run 'deck' to open the cockpit, or 'deck --help' for commands."
