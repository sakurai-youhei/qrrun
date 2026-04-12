#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:?VERSION is required}"
WINGET_PKGS_PAT="${WINGET_PKGS_PAT:?WINGET_PKGS_PAT is required}"
SOURCE_REPO="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"

WINGET_FORK_REPO="${WINGET_FORK_REPO:-sakurai-youhei/winget-pkgs}"
WINGET_UPSTREAM_REPO="${WINGET_UPSTREAM_REPO:-microsoft/winget-pkgs}"
PACKAGE_IDENTIFIER="sakurai-youhei.qrrun"
PACKAGE_PUBLISHER_DIR="sakurai-youhei"
PACKAGE_NAME_DIR="qrrun"

VERSION_NO_V="${VERSION#v}"
if [[ "${VERSION_NO_V}" =~ -(alpha|beta|rc)(\.|$) ]]; then
  echo "Skipping winget-pkgs PR for pre-release tag: ${VERSION}"
  exit 0
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

BASE_RELEASE_URL="https://github.com/${SOURCE_REPO}/releases/download/${VERSION}"
X64_URL="${BASE_RELEASE_URL}/qrrun_${VERSION}_windows_amd64.msi"
ARM64_URL="${BASE_RELEASE_URL}/qrrun_${VERSION}_windows_arm64.msi"

echo "Downloading MSI assets for ${VERSION}"
curl -fsSL -o "${TMP_DIR}/qrrun_x64.msi" "${X64_URL}"
curl -fsSL -o "${TMP_DIR}/qrrun_arm64.msi" "${ARM64_URL}"

X64_SHA256="$(sha256sum "${TMP_DIR}/qrrun_x64.msi" | awk '{print toupper($1)}')"
ARM64_SHA256="$(sha256sum "${TMP_DIR}/qrrun_arm64.msi" | awk '{print toupper($1)}')"
RELEASE_DATE="$(date -u +%F)"

echo "Cloning ${WINGET_FORK_REPO}"
git clone "https://x-access-token:${WINGET_PKGS_PAT}@github.com/${WINGET_FORK_REPO}.git" "${TMP_DIR}/winget-pkgs"

cd "${TMP_DIR}/winget-pkgs"
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

BRANCH_NAME="sakurai-youhei-qrrun-${VERSION_NO_V}"
if git ls-remote --exit-code --heads origin "${BRANCH_NAME}" >/dev/null 2>&1; then
  git switch --track "origin/${BRANCH_NAME}"
else
  git switch -c "${BRANCH_NAME}"
fi

MANIFEST_DIR="manifests/s/${PACKAGE_PUBLISHER_DIR}/${PACKAGE_NAME_DIR}/${VERSION_NO_V}"
mkdir -p "${MANIFEST_DIR}"

cat >"${MANIFEST_DIR}/${PACKAGE_IDENTIFIER}.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.version.1.12.0.schema.json

PackageIdentifier: ${PACKAGE_IDENTIFIER}
PackageVersion: ${VERSION_NO_V}
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.12.0
EOF

cat >"${MANIFEST_DIR}/${PACKAGE_IDENTIFIER}.locale.en-US.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.defaultLocale.1.12.0.schema.json

PackageIdentifier: ${PACKAGE_IDENTIFIER}
PackageVersion: ${VERSION_NO_V}
PackageLocale: en-US
Publisher: sakurai-youhei
PublisherUrl: https://github.com/sakurai-youhei
PublisherSupportUrl: https://github.com/sakurai-youhei/qrrun/issues
PackageName: qrrun
PackageUrl: https://github.com/sakurai-youhei/qrrun
License: MIT
LicenseUrl: https://github.com/sakurai-youhei/qrrun/blob/${VERSION}/LICENSE
ShortDescription: Tunnel local code and run it via QR.
Description: qrrun tunnels local scripts and lets mobile runtimes execute them by scanning a QR code.
Moniker: qrrun
Tags:
  - qr
  - tunnel
  - pythonista
  - cloudflared
ReleaseNotesUrl: https://github.com/sakurai-youhei/qrrun/releases/tag/${VERSION}
ManifestType: defaultLocale
ManifestVersion: 1.12.0
EOF

cat >"${MANIFEST_DIR}/${PACKAGE_IDENTIFIER}.installer.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.installer.1.12.0.schema.json

PackageIdentifier: ${PACKAGE_IDENTIFIER}
PackageVersion: ${VERSION_NO_V}
InstallerLocale: en-US
InstallerType: wix
Scope: machine
UpgradeBehavior: install
Commands:
  - qrrun
AppsAndFeaturesEntries:
  - UpgradeCode: '{B1A2C8E2-3E4B-4F93-ABF7-D39C45FB0C6D}'
ReleaseDate: ${RELEASE_DATE}
Installers:
  - Architecture: x64
    InstallerUrl: ${X64_URL}
    InstallerSha256: ${X64_SHA256}
  - Architecture: arm64
    InstallerUrl: ${ARM64_URL}
    InstallerSha256: ${ARM64_SHA256}
ManifestType: installer
ManifestVersion: 1.12.0
EOF

if git diff --quiet -- "${MANIFEST_DIR}"; then
  echo "No manifest changes detected. Skipping commit and PR creation."
  exit 0
fi

git add "${MANIFEST_DIR}"
git commit -m "Add ${PACKAGE_IDENTIFIER} version ${VERSION_NO_V}"
git push --set-upstream origin "${BRANCH_NAME}"

export GH_TOKEN="${WINGET_PKGS_PAT}"
EXISTING_PR_NUMBER="$(gh pr list --repo "${WINGET_UPSTREAM_REPO}" --head "sakurai-youhei:${BRANCH_NAME}" --state open --json number --jq '.[0].number')"
if [[ -n "${EXISTING_PR_NUMBER}" && "${EXISTING_PR_NUMBER}" != "null" ]]; then
  echo "winget-pkgs PR already exists: #${EXISTING_PR_NUMBER}"
  exit 0
fi

PR_TEMPLATE_FILE=".github/PULL_REQUEST_TEMPLATE.md"
if [[ ! -f "${PR_TEMPLATE_FILE}" ]]; then
  echo "PR template file not found: ${PR_TEMPLATE_FILE}"
  exit 1
fi

NEW_PR_URL="$(gh pr create \
  --repo "${WINGET_UPSTREAM_REPO}" \
  --base master \
  --head "sakurai-youhei:${BRANCH_NAME}" \
  --title "Add ${PACKAGE_IDENTIFIER} version ${VERSION_NO_V}" \
  --body-file "${PR_TEMPLATE_FILE}")"

NEW_PR_NUMBER="${NEW_PR_URL##*/}"
if [[ ! "${NEW_PR_NUMBER}" =~ ^[0-9]+$ ]]; then
  echo "Failed to parse PR number from URL: ${NEW_PR_URL}"
  exit 1
fi

gh pr comment \
  --repo "${WINGET_UPSTREAM_REPO}" \
  "${NEW_PR_NUMBER}" \
  --body "@sakurai-youhei winget-pkgs PR is open. Please review and complete the checklist manually."
