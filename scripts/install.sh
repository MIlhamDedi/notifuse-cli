#!/usr/bin/env bash
set -euo pipefail

repo="milhamdedi/notifuse-cli"
version="${NOTIFUSE_CLI_VERSION:-latest}"
bin_dir="${BIN_DIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  arm64|aarch64) arch="aarch64" ;;
  x86_64|amd64) arch="x86_64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

if [[ "$version" == "latest" ]]; then
  version="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" | sed -n 's/.*"tag_name": *"\\([^"]*\\)".*/\\1/p' | head -n1)"
fi

archive="notifuse-cli-${version}-${os}-${arch}.tar.gz"
url="https://github.com/${repo}/releases/download/${version}/${archive}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "$url" -o "$tmp/$archive"
mkdir -p "$bin_dir"
tar -xzf "$tmp/$archive" -C "$bin_dir" notifuse
chmod +x "$bin_dir/notifuse"
echo "installed $bin_dir/notifuse"
