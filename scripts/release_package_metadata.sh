#!/usr/bin/env bash

readonly QRRUN_PACKAGE_NAME="${QRRUN_PACKAGE_NAME:-qrrun}"
readonly QRRUN_PACKAGE_MAINTAINER="${QRRUN_PACKAGE_MAINTAINER:-Youhei Sakurai}"
readonly QRRUN_PACKAGE_SOURCE_REPO="${QRRUN_PACKAGE_SOURCE_REPO:-${REPO:-sakurai-youhei/qrrun}}"
readonly QRRUN_PACKAGE_HOMEPAGE="${QRRUN_PACKAGE_HOMEPAGE:-https://github.com/${QRRUN_PACKAGE_SOURCE_REPO}}"
readonly QRRUN_PACKAGE_LICENSE="${QRRUN_PACKAGE_LICENSE:-MIT}"
readonly QRRUN_PACKAGE_TAGLINE="${QRRUN_PACKAGE_TAGLINE:-Prototype locally, run on your phone via a QR and a quick tunnel}"
readonly QRRUN_PACKAGE_DESCRIPTION="${QRRUN_PACKAGE_DESCRIPTION:-qrrun tunnels local scripts and lets mobile runtimes execute them by scanning a QR code.}"
