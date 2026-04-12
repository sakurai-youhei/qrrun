#!/usr/bin/env bash
set -euo pipefail

REPO="sakurai-youhei/qrrun"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

usage() {
  cat <<'EOF'
Install qrrun from GitHub Releases.

Usage:
  scripts/install.sh [version]

Examples:
  scripts/install.sh
  scripts/install.sh v0.1.0

Environment variables:
  INSTALL_DIR   Install destination directory (default: /usr/local/bin)
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

version="${1:-latest}"

if [[ "$version" == "latest" ]]; then
  version="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | head -n1)"
fi

if [[ -z "$version" ]]; then
  echo "Failed to resolve release version." >&2
  exit 1
fi

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="qrrun_${version}_${os}_${arch}.tar.gz"
bin_name="qrrun_${version}_${os}_${arch}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

curl -fsSL "https://github.com/${REPO}/releases/download/${version}/${asset}" -o "${tmp_dir}/${asset}"
tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir"

if [[ ! -f "${tmp_dir}/${bin_name}" ]]; then
  echo "Expected binary not found in archive: ${bin_name}" >&2
  exit 1
fi

if [[ ! -d "$INSTALL_DIR" ]]; then
  echo "Install directory does not exist: $INSTALL_DIR" >&2
  exit 1
fi

if [[ -w "$INSTALL_DIR" ]]; then
  install -m 0755 "${tmp_dir}/${bin_name}" "$INSTALL_DIR/qrrun"
else
  sudo install -m 0755 "${tmp_dir}/${bin_name}" "$INSTALL_DIR/qrrun"
fi

echo "Installed qrrun to ${INSTALL_DIR}/qrrun"
"${INSTALL_DIR}/qrrun" --version
