#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
install_dir="${INSTALL_DIR:-/usr/local/bin}"
binary_name="${BINARY_NAME:-om-gpx}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to build om-gpx, but it was not found in PATH" >&2
  exit 1
fi

echo "Building $binary_name..."
(
  cd "$repo_dir"
  go build -o "$tmp_dir/$binary_name" .
)

echo "Installing to $install_dir/$binary_name..."
mkdir -p "$install_dir" 2>/dev/null || sudo mkdir -p "$install_dir"
if [ -w "$install_dir" ]; then
  install -m 0755 "$tmp_dir/$binary_name" "$install_dir/$binary_name"
else
  sudo install -m 0755 "$tmp_dir/$binary_name" "$install_dir/$binary_name"
fi

echo "Installed $binary_name"
echo "Try: $binary_name /path/to/OI.Share/log-folder"
