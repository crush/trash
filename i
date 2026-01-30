#!/bin/sh
set -e

REPO="crush/snap"
BIN="$HOME/.local/bin"

ARCH=$(uname -m)
case "$ARCH" in
  arm64) ARCH="arm64" ;;
  x86_64) ARCH="amd64" ;;
  *) echo "unsupported: $ARCH" >&2; exit 1 ;;
esac

VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
[ -z "$VERSION" ] && exit 1

mkdir -p "$BIN"

TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

curl -fsSL "https://github.com/$REPO/releases/download/$VERSION/snap_darwin_$ARCH.tar.gz" | tar -xz -C "$TMP"
mv "$TMP/snap" "$BIN/snap"

case ":$PATH:" in
  *":$BIN:"*) ;;
  *) echo "add to path: export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac
