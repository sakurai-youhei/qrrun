#!/usr/bin/env bash
set -euo pipefail

DEB_VERSION="${VERSION#v}"
RPM_ARCH="$GOARCH"
if [ "$GOARCH" = "amd64" ]; then
  RPM_ARCH="x86_64"
elif [ "$GOARCH" = "arm64" ]; then
  RPM_ARCH="aarch64"
fi

RPM_VERSION="${DEB_VERSION%%-*}"
RPM_RELEASE="1"
if [ "${DEB_VERSION}" != "${RPM_VERSION}" ]; then
  RPM_RELEASE="0.$(echo "${DEB_VERSION#"${RPM_VERSION}"-}" | tr -c '[:alnum:].' '.' | sed 's/[.]\+$//')"
fi

fpm \
  -s dir \
  -t rpm \
  -n qrrun \
  -v "${RPM_VERSION}" \
  --iteration "${RPM_RELEASE}" \
  --architecture "${RPM_ARCH}" \
  --license MIT \
  --description "Tunnel local code and run it via QR." \
  --package "dist/qrrun_${DEB_VERSION}_${RPM_ARCH}.rpm" \
  "dist/${BIN}=/usr/bin/qrrun"
