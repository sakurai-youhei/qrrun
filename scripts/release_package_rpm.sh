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

require_tool fpm

if [[ ! -f "dist/${BIN}" ]]; then
  log "Binary not found: dist/${BIN}"
  exit 1
fi

PKG_VERSION="${VERSION#v}"
RPM_ARCH="${GOARCH}"
if [[ "${GOARCH}" == "amd64" ]]; then
  RPM_ARCH="x86_64"
elif [[ "${GOARCH}" == "arm64" ]]; then
  RPM_ARCH="aarch64"
else
  log "Unsupported GOARCH for RPM packaging: ${GOARCH}"
  exit 1
fi

RPM_VERSION="${PKG_VERSION%%-*}"
RPM_RELEASE="1"
if [[ "${PKG_VERSION}" != "${RPM_VERSION}" ]]; then
  RPM_RELEASE="0.$(echo "${PKG_VERSION#"${RPM_VERSION}"-}" | tr -c '[:alnum:].' '.' | sed 's/[.]\+$//')"
fi

log "Building RPM package for ${RPM_ARCH} (version ${RPM_VERSION}-${RPM_RELEASE})"
fpm \
  -s dir \
  -t rpm \
  -n qrrun \
  -v "${RPM_VERSION}" \
  --iteration "${RPM_RELEASE}" \
  --architecture "${RPM_ARCH}" \
  --license MIT \
  --description "Prototype locally, run on your phone via a QR and a quick tunnel." \
  --package "dist/qrrun_${PKG_VERSION}_${RPM_ARCH}.rpm" \
  "dist/${BIN}=/usr/bin/qrrun"
log "RPM package built successfully"
