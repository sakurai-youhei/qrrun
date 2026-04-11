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
  RPM_RELEASE="0.$(echo "${DEB_VERSION#${RPM_VERSION}-}" | tr -c '[:alnum:].' '.')"
fi

RPM_TOPDIR="$PWD/dist/rpmbuild"
mkdir -p "${RPM_TOPDIR}/BUILD" "${RPM_TOPDIR}/RPMS" "${RPM_TOPDIR}/SOURCES" "${RPM_TOPDIR}/SPECS" "${RPM_TOPDIR}/SRPMS"
cp "dist/${BIN}" "${RPM_TOPDIR}/SOURCES/qrrun"

cat > "${RPM_TOPDIR}/SPECS/qrrun.spec" <<EOF
Name:           qrrun
Version:        ${RPM_VERSION}
Release:        ${RPM_RELEASE}
Summary:        Tunnel local code and run it via QR
License:        MIT
BuildArch:      ${RPM_ARCH}
Source0:        qrrun

%description
Tunnel local code and run it via QR.

%prep

%build

%install
mkdir -p %{buildroot}/usr/bin
install -m 0755 %{SOURCE0} %{buildroot}/usr/bin/qrrun

%files
/usr/bin/qrrun

%changelog
* $(date -u '+%a %b %d %Y') qrrun maintainers - ${DEB_VERSION}
- Automated release build
EOF

rpmbuild \
  --target "${RPM_ARCH}" \
  --define "_topdir ${RPM_TOPDIR}" \
  -bb "${RPM_TOPDIR}/SPECS/qrrun.spec"
find "${RPM_TOPDIR}/RPMS" -type f -name '*.rpm' -exec cp {} dist/ \;
