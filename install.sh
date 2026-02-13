#!/usr/bin/env bash
set -euo pipefail

REPO="rendis/geotap"
BINARY="geotap"
INSTALL_DIR="/usr/local/bin"

info()  { printf "\033[1;34m==>\033[0m %s\n" "$*"; }
error() { printf "\033[1;31merror:\033[0m %s\n" "$*" >&2; exit 1; }

# Detect OS
case "$(uname -s)" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *) error "Unsupported OS: $(uname -s)" ;;
esac

# Detect architecture
case "$(uname -m)" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) error "Unsupported architecture: $(uname -m)" ;;
esac

info "Detected platform: ${OS}/${ARCH}"

# Get latest version from GitHub API
info "Fetching latest version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  error "Could not determine latest version"
fi
info "Latest version: v${VERSION}"

# Build download URL
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
  EXT="zip"
fi
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

# Download
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${URL}..."
curl -fsSL -o "${TMPDIR}/${FILENAME}" "$URL" || error "Download failed. Check that the release exists."

# Extract
info "Extracting..."
if [ "$EXT" = "tar.gz" ]; then
  tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"
else
  unzip -q "${TMPDIR}/${FILENAME}" -d "$TMPDIR"
fi

# Install
info "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi
chmod +x "${INSTALL_DIR}/${BINARY}"

# Verify
INSTALLED_VERSION=$("${INSTALL_DIR}/${BINARY}" version 2>/dev/null || true)
info "Installed: ${INSTALLED_VERSION}"
info "Run 'geotap' to start"
