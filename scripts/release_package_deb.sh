#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:?VERSION is required}"
GOARCH="${GOARCH:?GOARCH is required}"
BIN="${BIN:?BIN is required}"

SCRIPT_NAME="$(basename "$0")"
log() {
  echo "[${SCRIPT_NAME}] $*"
}

require_tool() {
  local tool="$1"
  if ! command -v "${tool}" >/dev/null 2>&1; then
    log "Required tool is not installed: ${tool}"
    exit 1
  fi
}

require_tool dpkg-deb
require_tool install

if [[ ! -f "dist/${BIN}" ]]; then
  log "Binary not found: dist/${BIN}"
  exit 1
fi

DEB_ARCH="${GOARCH}"
DEB_VERSION="${VERSION#v}"
DEB_BASE="qrrun_${DEB_VERSION}_${DEB_ARCH}"
PKG_ROOT="dist/${DEB_BASE}"

log "Preparing Debian package layout: ${PKG_ROOT}"
mkdir -p "${PKG_ROOT}/DEBIAN" "${PKG_ROOT}/usr/bin"
install -m 0755 "dist/${BIN}" "${PKG_ROOT}/usr/bin/qrrun"

cat >"${PKG_ROOT}/DEBIAN/control" <<EOF
Package: qrrun
Version: ${DEB_VERSION}
Section: utils
Priority: optional
Architecture: ${DEB_ARCH}
Maintainer: qrrun maintainers
Description: Tunnel local code and run it via QR
EOF

log "Building Debian package: dist/${DEB_BASE}.deb"
dpkg-deb --build --root-owner-group "${PKG_ROOT}" "dist/${DEB_BASE}.deb"
log "Debian package built successfully"
