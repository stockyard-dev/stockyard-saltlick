#!/bin/sh
set -e

REPO="stockyard-dev/stockyard-saltlick"
BINARY="saltlick"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

VERSION=$(curl -sf "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
if [ -z "$VERSION" ]; then
  echo "Could not determine latest version"
  exit 1
fi

URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY}_${OS}_${ARCH}"
echo "Downloading $BINARY $VERSION for $OS/$ARCH..."
curl -sfL "$URL" -o "/tmp/$BINARY"
chmod +x "/tmp/$BINARY"

if [ -w "$INSTALL_DIR" ]; then
  mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
else
  sudo mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "$BINARY $VERSION installed to $INSTALL_DIR/$BINARY"
echo ""
echo "Start with:"
echo "  $BINARY"
echo ""
echo "Dashboard: http://localhost:8800/ui"
