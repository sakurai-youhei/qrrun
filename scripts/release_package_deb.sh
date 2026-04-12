#!/usr/bin/env bash
set -euo pipefail

DEB_ARCH="$GOARCH"
DEB_VERSION="${VERSION#v}"
DEB_BASE="qrrun_${DEB_VERSION}_${DEB_ARCH}"
PKG_ROOT="dist/${DEB_BASE}"

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

dpkg-deb --build --root-owner-group "${PKG_ROOT}" "dist/${DEB_BASE}.deb"
