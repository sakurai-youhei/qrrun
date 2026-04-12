#!/usr/bin/env bash
set -euo pipefail

FORMULA_VERSION="${VERSION#v}"
DARWIN_AMD64="qrrun_${VERSION}_darwin_amd64.tar.gz"
DARWIN_ARM64="qrrun_${VERSION}_darwin_arm64.tar.gz"

SHA_AMD64="$(sha256sum "dist/${DARWIN_AMD64}" | awk '{print $1}')"
SHA_ARM64="$(sha256sum "dist/${DARWIN_ARM64}" | awk '{print $1}')"

mkdir -p dist/homebrew

cat >dist/homebrew/qrrun.rb <<EOF
class Qrrun < Formula
  desc "Tunnel local code and run it via QR"
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
      bin.install "qrrun_#{version}_darwin_amd64" => "qrrun"
    else
      bin.install "qrrun_#{version}_darwin_arm64" => "qrrun"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/qrrun --version")
  end
end
EOF
