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
  linux)   SUFFIX="linux-${ARCH}" ;;
  darwin)  SUFFIX="darwin-${ARCH}" ;;
  *)       echo "Unsupported OS: $OS. macOS and Linux only."
           exit 1 ;;
esac

# Helper for HTTP requests
fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$1"
  else
    echo "Error: curl or wget required"
    exit 1
  fi
}

fetch_file() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  else
    wget -qO "$2" "$1"
  fi
}

# Get latest release tag
TAG=$(fetch "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)

if [ -z "$TAG" ]; then
  echo "Error: could not determine latest release"
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/buff-er-${SUFFIX}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"
echo "Downloading buff-er ${TAG} for ${OS}/${ARCH}..."

TMP=$(mktemp)
trap 'rm -f "$TMP" "$TMP.checksums"' EXIT

fetch_file "$URL" "$TMP"

# Verify checksum
if command -v sha256sum >/dev/null 2>&1; then
  fetch "$CHECKSUM_URL" > "$TMP.checksums"
  EXPECTED=$(grep "buff-er-${SUFFIX}$" "$TMP.checksums" | awk '{print $1}')
  ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
  if [ -n "$EXPECTED" ] && [ "$EXPECTED" != "$ACTUAL" ]; then
    echo "Error: checksum mismatch"
    echo "  expected: $EXPECTED"
    echo "  got:      $ACTUAL"
    exit 1
  fi
elif command -v shasum >/dev/null 2>&1; then
  fetch "$CHECKSUM_URL" > "$TMP.checksums"
  EXPECTED=$(grep "buff-er-${SUFFIX}$" "$TMP.checksums" | awk '{print $1}')
  ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
  if [ -n "$EXPECTED" ] && [ "$EXPECTED" != "$ACTUAL" ]; then
    echo "Error: checksum mismatch"
    echo "  expected: $EXPECTED"
    echo "  got:      $ACTUAL"
    exit 1
  fi
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
