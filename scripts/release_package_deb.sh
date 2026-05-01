#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:?VERSION is required}"
GOARCH="${GOARCH:?GOARCH is required}"
BIN="${BIN:?BIN is required}"

SCRIPT_NAME="$(basename "$0")"
SCRIPT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=./release_package_metadata.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/release_package_metadata.sh"

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
DEB_BASE="${QRRUN_PACKAGE_NAME}_${DEB_VERSION}_${DEB_ARCH}"
PKG_ROOT="dist/${DEB_BASE}"

log "Preparing Debian package layout: ${PKG_ROOT}"
mkdir -p "${PKG_ROOT}/DEBIAN" "${PKG_ROOT}/usr/bin"
install -m 0755 "dist/${BIN}" "${PKG_ROOT}/usr/bin/${QRRUN_PACKAGE_NAME}"

cat >"${PKG_ROOT}/DEBIAN/control" <<EOF
Package: ${QRRUN_PACKAGE_NAME}
Version: ${DEB_VERSION}
Section: utils
Priority: optional
Architecture: ${DEB_ARCH}
Maintainer: ${QRRUN_PACKAGE_MAINTAINER}
Homepage: ${QRRUN_PACKAGE_HOMEPAGE}
Description: ${QRRUN_PACKAGE_TAGLINE}
 ${QRRUN_PACKAGE_DESCRIPTION}
EOF

log "Building Debian package: dist/${DEB_BASE}.deb"
dpkg-deb --build --root-owner-group "${PKG_ROOT}" "dist/${DEB_BASE}.deb"
log "Debian package built successfully"
