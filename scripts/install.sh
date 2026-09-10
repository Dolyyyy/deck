#!/usr/bin/env bash
# ==============================================================================
#  DECK - Terminal Infrastructure Cockpit Installer
#  Repository: https://github.com/Dolyyyy/deck
# ==============================================================================

set -euo pipefail

# ANSI Colors
BOLD="\033[1m"
DIM="\033[2m"
CYAN="\033[38;2;137;180;250m"
GREEN="\033[38;2;166;227;161m"
YELLOW="\033[38;2;249;226;175m"
RED="\033[38;2;243;139;168m"
RESET="\033[0m"

REPO="Dolyyyy/deck"
BINARY_NAME="deck"
FALLBACK_VERSION="v0.2.0"

print_banner() {
    cat << "EOF"
  ____  _____ ____ _  __
 / __ \/ ___// __ \/ / /
/ /_/ / /__ / /_/ / /_/ 
\____/\___// .___/\__,_/  Terminal Cockpit
          /_/
EOF
}

echo -e "\n${CYAN}$(print_banner)${RESET}\n"
echo -e "${BOLD}⚡ Installing Deck Terminal Infrastructure Cockpit...${RESET}\n"

# 1. Detect OS & Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7*|armhf) ARCH="arm" ;;
    *) echo -e "${RED}❌ Unsupported CPU architecture: ${ARCH}${RESET}" && exit 1 ;;
esac

case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo -e "${RED}❌ Unsupported operating system: ${OS}${RESET}" && exit 1 ;;
esac

echo -e " ${GREEN}✔${RESET} Detected platform: ${BOLD}${OS}/${ARCH}${RESET}"

# 2. Fetch Latest Release Version
echo -e " ${CYAN}◌${RESET} Resolving release version..."
RELEASE_TAG=""
if command -v curl >/dev/null 2>&1; then
    RELEASE_TAG=$(curl -sSL -H "Accept: application/vnd.github.v3+json" "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | head -n 1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/' || true)
fi

if [ -z "$RELEASE_TAG" ]; then
    RELEASE_TAG="$FALLBACK_VERSION"
fi

VERSION_NUM="${RELEASE_TAG#v}"
TARBALL="deck_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${RELEASE_TAG}/${TARBALL}"

echo -e " ${GREEN}✔${RESET} Target release: ${BOLD}${RELEASE_TAG}${RESET}"

TEMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# 3. Download or Compile
DOWNLOADED=0
echo -e " ${CYAN}◌${RESET} Downloading pre-built archive from GitHub Releases..."
if curl -fsSL "$DOWNLOAD_URL" -o "${TEMP_DIR}/${TARBALL}" 2>/dev/null; then
    if tar -xzf "${TEMP_DIR}/${TARBALL}" -C "$TEMP_DIR" 2>/dev/null; then
        if [ -f "${TEMP_DIR}/${BINARY_NAME}" ]; then
            DOWNLOADED=1
            echo -e " ${GREEN}✔${RESET} Downloaded and unpacked release binaries."
        fi
    fi
fi

if [ "$DOWNLOADED" -eq 0 ]; then
    echo -e " ${YELLOW}▲ Pre-compiled archive not found on releases. Building from source via Go...${RESET}"
    if command -v go >/dev/null 2>&1; then
        export CGO_ENABLED=0
        go install "github.com/${REPO}/cmd/deck@latest"
        GOPATH_BIN="$(go env GOPATH)/bin/${BINARY_NAME}"
        if [ -f "$GOPATH_BIN" ]; then
            cp "$GOPATH_BIN" "${TEMP_DIR}/${BINARY_NAME}"
            echo -e " ${GREEN}✔${RESET} Successfully built from source!"
        else
            echo -e "${RED}❌ Failed to build Deck binary from source.${RESET}"
            exit 1
        fi
    else
        echo -e "${RED}❌ Neither pre-built archive nor Go compiler are available.${RESET}"
        exit 1
    fi
fi

# 4. Install Binary to System PATH
INSTALL_DIR=""
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
    INSTALL_DIR="/usr/local/bin"
fi

if [ -z "$INSTALL_DIR" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

DEST_FILE="${INSTALL_DIR}/${BINARY_NAME}"
if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ ! -w "/usr/local/bin" ]; then
    sudo mv "${TEMP_DIR}/${BINARY_NAME}" "$DEST_FILE"
    sudo chmod +x "$DEST_FILE"
else
    mv "${TEMP_DIR}/${BINARY_NAME}" "$DEST_FILE"
    chmod +x "$DEST_FILE"
fi

# Also link into ~/.local/bin if installed in /usr/local/bin or vice-versa
mkdir -p "${HOME}/.local/bin"
if [ "$INSTALL_DIR" = "/usr/local/bin" ]; then
    cp -f "$DEST_FILE" "${HOME}/.local/bin/${BINARY_NAME}" 2>/dev/null || true
fi

# Ensure user PATH contains INSTALL_DIR
PATH_ADDED=0
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo -e " ${YELLOW}▲ Adding ${INSTALL_DIR} to PATH in shell profile...${RESET}"
        if [ -f "${HOME}/.bashrc" ]; then
            echo "export PATH=\"${INSTALL_DIR}:\$PATH\"" >> "${HOME}/.bashrc"
            PATH_ADDED=1
        fi
        if [ -f "${HOME}/.zshrc" ]; then
            echo "export PATH=\"${INSTALL_DIR}:\$PATH\"" >> "${HOME}/.zshrc"
            PATH_ADDED=1
        fi
        ;;
esac

echo -e " ${GREEN}✔${RESET} Installed Deck executable to ${BOLD}${DEST_FILE}${RESET}"

# 5. Verification
if [ -x "$DEST_FILE" ]; then
    INSTALLED_VER="$("$DEST_FILE" --version 2>/dev/null || echo "$RELEASE_TAG")"
    echo -e " ${GREEN}✔${RESET} Verified executable: ${BOLD}${INSTALLED_VER}${RESET}\n"
fi

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo -e "${BOLD}🎉 Deck has been successfully installed!${RESET}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo -e " 👉 Launch Cockpit : ${BOLD}${CYAN}deck${RESET}"
echo -e " 👉 Command List    : ${BOLD}${CYAN}deck --help${RESET}"
echo -e " 👉 Add Shell Jump  : ${BOLD}${CYAN}echo 'eval \"\$(deck init bash)\"' >> ~/.bashrc${RESET}"
if [ "$PATH_ADDED" -eq 1 ]; then
    echo -e "\n ${YELLOW}Note: Restart your terminal or run 'source ~/.bashrc' (or ~/.zshrc) to refresh PATH.${RESET}"
fi
echo ""
