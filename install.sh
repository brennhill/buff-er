#!/bin/sh
set -e

REPO="brennhill/buff-er"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux)  SUFFIX="linux-${ARCH}" ;;
  darwin) SUFFIX="darwin-${ARCH}" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest release tag
if command -v curl >/dev/null 2>&1; then
  TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
elif command -v wget >/dev/null 2>&1; then
  TAG=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
else
  echo "Error: curl or wget required"
  exit 1
fi

if [ -z "$TAG" ]; then
  echo "Error: could not determine latest release"
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/buff-er-${SUFFIX}"
echo "Downloading buff-er ${TAG} for ${OS}/${ARCH}..."

TMP=$(mktemp)
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TMP"
else
  wget -qO "$TMP" "$URL"
fi

chmod +x "$TMP"

# Install — use sudo if needed
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/buff-er"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$TMP" "${INSTALL_DIR}/buff-er"
fi

echo "buff-er ${TAG} installed to ${INSTALL_DIR}/buff-er"
echo ""
echo "Next: run 'buff-er install' to register hooks with Claude Code"
