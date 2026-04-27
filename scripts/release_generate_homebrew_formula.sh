#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:?VERSION is required}"
REPO="${REPO:?REPO is required}"

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

require_tool sha256sum
require_tool awk

FORMULA_VERSION="${VERSION#v}"
DARWIN_AMD64="qrrun_${VERSION}_darwin_amd64.tar.gz"
DARWIN_ARM64="qrrun_${VERSION}_darwin_arm64.tar.gz"

if [[ ! -f "dist/${DARWIN_AMD64}" ]]; then
  log "Missing release asset: dist/${DARWIN_AMD64}"
  exit 1
fi

if [[ ! -f "dist/${DARWIN_ARM64}" ]]; then
  log "Missing release asset: dist/${DARWIN_ARM64}"
  exit 1
fi

SHA_AMD64="$(sha256sum "dist/${DARWIN_AMD64}" | awk '{print $1}')"
SHA_ARM64="$(sha256sum "dist/${DARWIN_ARM64}" | awk '{print $1}')"

mkdir -p dist/homebrew

log "Generating Homebrew formula at dist/homebrew/qrrun.rb"

cat >dist/homebrew/qrrun.rb <<EOF
class Qrrun < Formula
  desc "Prototype locally, run on your phone via a QR and a quick tunnel"
  homepage "https://github.com/${REPO}"
  version "${FORMULA_VERSION}"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/${REPO}/releases/download/${VERSION}/${DARWIN_AMD64}"
      sha256 "${SHA_AMD64}"
    end

    if Hardware::CPU.arm?
      url "https://github.com/${REPO}/releases/download/${VERSION}/${DARWIN_ARM64}"
      sha256 "${SHA_ARM64}"
    end
  end

  def install
    if Hardware::CPU.intel?
      bin.install "qrrun_v#{version}_darwin_amd64" => "qrrun"
    else
      bin.install "qrrun_v#{version}_darwin_arm64" => "qrrun"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/qrrun --version")
  end
end
EOF

log "Homebrew formula generated successfully"
