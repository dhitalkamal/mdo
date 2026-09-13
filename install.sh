#!/bin/sh
# mdo installer - downloads the latest release binary for your OS/arch.
# usage: curl -fsSL https://raw.githubusercontent.com/dhitalkamal/mdo/main/install.sh | sh
# override install dir with BINDIR=/usr/local/bin
set -e

repo="dhitalkamal/mdo"
bindir="${BINDIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "mdo: unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "mdo: unsupported OS: $os" >&2; exit 1 ;;
esac

ver="$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" \
  | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)"
[ -n "$ver" ] || { echo "mdo: could not determine latest version" >&2; exit 1; }

url="https://github.com/$repo/releases/download/$ver/mdo_${ver}_${os}_${arch}.tar.gz"
echo "mdo: downloading $ver ($os/$arch)..."

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "$url" | tar -xz -C "$tmp"
mkdir -p "$bindir"
install -m 755 "$tmp/mdo" "$bindir/mdo"

echo "mdo: installed to $bindir/mdo"
case ":$PATH:" in
  *":$bindir:"*) ;;
  *) echo "mdo: add $bindir to your PATH to run 'mdo' directly" ;;
esac
