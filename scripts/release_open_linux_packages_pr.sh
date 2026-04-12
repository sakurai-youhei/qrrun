#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:?VERSION is required}"
LINUX_PACKAGES_PAT="${LINUX_PACKAGES_PAT:?LINUX_PACKAGES_PAT is required}"
QRRUN_GITHUB_TOKEN="${QRRUN_GITHUB_TOKEN:?QRRUN_GITHUB_TOKEN is required}"
LINUX_REPO_GPG_PRIVATE_KEY="${LINUX_REPO_GPG_PRIVATE_KEY:?LINUX_REPO_GPG_PRIVATE_KEY is required}"
LINUX_REPO_GPG_PASSPHRASE="${LINUX_REPO_GPG_PASSPHRASE:?LINUX_REPO_GPG_PASSPHRASE is required}"
LINUX_REPO_GPG_KEY_ID="${LINUX_REPO_GPG_KEY_ID:-}"
SOURCE_REPO="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"

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

gh_packages() {
  GH_TOKEN="${LINUX_PACKAGES_PAT}" gh "$@"
}

gh_source() {
  GH_TOKEN="${QRRUN_GITHUB_TOKEN}" gh "$@"
}

SOURCE_OWNER="${SOURCE_REPO%%/*}"
LINUX_PACKAGES_REPO="${LINUX_PACKAGES_REPO:-${SOURCE_OWNER}/linux-packages}"
PACKAGES_OWNER="${LINUX_PACKAGES_REPO%%/*}"

VERSION_NO_V="${VERSION#v}"
if [[ "${VERSION_NO_V}" =~ -(alpha|beta|rc)(\.|$) ]]; then
  log "Skipping linux-packages PR for pre-release tag: ${VERSION}"
  exit 0
fi

for tool in gh git curl gpg dpkg-scanpackages apt-ftparchive createrepo_c; do
  require_tool "${tool}"
done

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

export GNUPGHOME="${TMP_DIR}/gnupg"
mkdir -p "${GNUPGHOME}"
chmod 700 "${GNUPGHOME}"

printf '%s' "${LINUX_REPO_GPG_PRIVATE_KEY}" | gpg --batch --import

SIGNING_KEY="${LINUX_REPO_GPG_KEY_ID}"
if [[ -z "${SIGNING_KEY}" ]]; then
  SIGNING_KEY="$(gpg --batch --list-secret-keys --with-colons | awk -F: '$1 == "sec" {print $5; exit}')"
fi

if [[ -z "${SIGNING_KEY}" ]]; then
  log "Failed to resolve signing key ID from imported private key."
  exit 1
fi

gpg_sign() {
  gpg \
    --batch \
    --yes \
    --pinentry-mode loopback \
    --passphrase "${LINUX_REPO_GPG_PASSPHRASE}" \
    --local-user "${SIGNING_KEY}" \
    "$@"
}

BASE_RELEASE_URL="https://github.com/${SOURCE_REPO}/releases/download/${VERSION}"
DEB_AMD64="qrrun_${VERSION_NO_V}_amd64.deb"
DEB_ARM64="qrrun_${VERSION_NO_V}_arm64.deb"
RPM_X86_64="qrrun_${VERSION_NO_V}_x86_64.rpm"
RPM_AARCH64="qrrun_${VERSION_NO_V}_aarch64.rpm"

log "Downloading Linux package assets for ${VERSION}"
curl -fsSL -o "${TMP_DIR}/${DEB_AMD64}" "${BASE_RELEASE_URL}/${DEB_AMD64}"
curl -fsSL -o "${TMP_DIR}/${DEB_ARM64}" "${BASE_RELEASE_URL}/${DEB_ARM64}"
curl -fsSL -o "${TMP_DIR}/${RPM_X86_64}" "${BASE_RELEASE_URL}/${RPM_X86_64}"
curl -fsSL -o "${TMP_DIR}/${RPM_AARCH64}" "${BASE_RELEASE_URL}/${RPM_AARCH64}"

if ! gh_packages repo view "${LINUX_PACKAGES_REPO}" >/dev/null 2>&1; then
  echo "::warning::Linux package repository is not accessible: ${LINUX_PACKAGES_REPO}. Skipping automation."
  exit 0
fi

DEFAULT_BRANCH="$(gh_packages repo view "${LINUX_PACKAGES_REPO}" --json defaultBranchRef --jq '.defaultBranchRef.name')"
if [[ -z "${DEFAULT_BRANCH}" || "${DEFAULT_BRANCH}" == "null" ]]; then
  DEFAULT_BRANCH="main"
fi
log "Using linux-packages default branch: ${DEFAULT_BRANCH}"

log "Cloning ${LINUX_PACKAGES_REPO}"
git clone "https://x-access-token:${LINUX_PACKAGES_PAT}@github.com/${LINUX_PACKAGES_REPO}.git" "${TMP_DIR}/linux-packages"

cd "${TMP_DIR}/linux-packages"
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

git switch "${DEFAULT_BRANCH}"

BRANCH_NAME="qrrun-${VERSION_NO_V}"
if git ls-remote --exit-code --heads origin "${BRANCH_NAME}" >/dev/null 2>&1; then
  git switch --track "origin/${BRANCH_NAME}"
else
  git switch -c "${BRANCH_NAME}"
fi

APT_POOL_DIR="apt/pool/main/q/qrrun"
APT_DIST_DIR="apt/dists/stable/main"
RPM_X86_DIR="rpm/x86_64"
RPM_AARCH64_DIR="rpm/aarch64"
KEYS_DIR="keys"

mkdir -p "${APT_POOL_DIR}" "${APT_DIST_DIR}/binary-amd64" "${APT_DIST_DIR}/binary-arm64"
mkdir -p "${RPM_X86_DIR}" "${RPM_AARCH64_DIR}"
mkdir -p "${KEYS_DIR}"

cp "${TMP_DIR}/${DEB_AMD64}" "${APT_POOL_DIR}/${DEB_AMD64}"
cp "${TMP_DIR}/${DEB_ARM64}" "${APT_POOL_DIR}/${DEB_ARM64}"
cp "${TMP_DIR}/${RPM_X86_64}" "${RPM_X86_DIR}/${RPM_X86_64}"
cp "${TMP_DIR}/${RPM_AARCH64}" "${RPM_AARCH64_DIR}/${RPM_AARCH64}"

pushd apt >/dev/null
rm -f dists/stable/main/binary-amd64/Packages dists/stable/main/binary-amd64/Packages.gz
rm -f dists/stable/main/binary-arm64/Packages dists/stable/main/binary-arm64/Packages.gz
rm -f dists/stable/InRelease dists/stable/Release.gpg

dpkg-scanpackages --arch amd64 pool/main/q/qrrun /dev/null >dists/stable/main/binary-amd64/Packages
gzip -9c dists/stable/main/binary-amd64/Packages >dists/stable/main/binary-amd64/Packages.gz

dpkg-scanpackages --arch arm64 pool/main/q/qrrun /dev/null >dists/stable/main/binary-arm64/Packages
gzip -9c dists/stable/main/binary-arm64/Packages >dists/stable/main/binary-arm64/Packages.gz

apt-ftparchive \
  -o APT::FTPArchive::Release::Origin="qrrun" \
  -o APT::FTPArchive::Release::Label="qrrun" \
  -o APT::FTPArchive::Release::Suite="stable" \
  -o APT::FTPArchive::Release::Codename="stable" \
  release dists/stable >dists/stable/Release

gpg_sign --clearsign --output dists/stable/InRelease dists/stable/Release
gpg_sign --detach-sign --armor --output dists/stable/Release.gpg dists/stable/Release
popd >/dev/null

createrepo_c --update --simple-md-filenames "${RPM_X86_DIR}"
createrepo_c --update --simple-md-filenames "${RPM_AARCH64_DIR}"

rm -f "${RPM_X86_DIR}/repodata/repomd.xml.asc" "${RPM_AARCH64_DIR}/repodata/repomd.xml.asc"
gpg_sign --detach-sign --armor --output "${RPM_X86_DIR}/repodata/repomd.xml.asc" "${RPM_X86_DIR}/repodata/repomd.xml"
gpg_sign --detach-sign --armor --output "${RPM_AARCH64_DIR}/repodata/repomd.xml.asc" "${RPM_AARCH64_DIR}/repodata/repomd.xml"

gpg --batch --yes --armor --output "${KEYS_DIR}/qrrun-packages.asc" --export "${SIGNING_KEY}"
gpg --batch --yes --output "${KEYS_DIR}/qrrun-packages.gpg" --export "${SIGNING_KEY}"

git add -A apt rpm keys

if git diff --cached --quiet; then
  log "No repository changes detected. Skipping commit and PR creation."
  exit 0
fi

git commit -m "Update qrrun Linux packages to ${VERSION}"
git push --set-upstream origin "${BRANCH_NAME}"

ISSUE_TITLE="Release: update Linux package repositories for qrrun ${VERSION_NO_V}"
EXISTING_ISSUE_NUMBER="$(gh_source issue list --repo "${SOURCE_REPO}" --state open --search "\"${ISSUE_TITLE}\" in:title" --json number --jq '.[0].number')"

if [[ -n "${EXISTING_ISSUE_NUMBER}" && "${EXISTING_ISSUE_NUMBER}" != "null" ]]; then
  ISSUE_URL="$(gh_source issue view "${EXISTING_ISSUE_NUMBER}" --repo "${SOURCE_REPO}" --json url --jq '.url')"
else
  ISSUE_BODY_FILE="${TMP_DIR}/qrrun-linux-release-issue.md"
  cat >"${ISSUE_BODY_FILE}" <<EOF
## Summary
- Track linux-packages submission for qrrun ${VERSION_NO_V}

## Release
- qrrun release tag: ${VERSION}
- source repo: https://github.com/${SOURCE_REPO}

## Linux Package Repositories
- repo: https://github.com/${LINUX_PACKAGES_REPO}
- target branch: ${BRANCH_NAME}
EOF

  ISSUE_URL="$(gh_source issue create --repo "${SOURCE_REPO}" --title "${ISSUE_TITLE}" --body-file "${ISSUE_BODY_FILE}")"
fi

EXISTING_PR_NUMBER="$(gh_packages pr list --repo "${LINUX_PACKAGES_REPO}" --head "${PACKAGES_OWNER}:${BRANCH_NAME}" --state open --json number --jq '.[0].number')"
if [[ -n "${EXISTING_PR_NUMBER}" && "${EXISTING_PR_NUMBER}" != "null" ]]; then
  gh_packages pr comment --repo "${LINUX_PACKAGES_REPO}" "${EXISTING_PR_NUMBER}" --body "qrrun tracking issue: ${ISSUE_URL}"
  log "Linux packages PR already exists: #${EXISTING_PR_NUMBER}"
  exit 0
fi

PR_BODY_FILE="${TMP_DIR}/linux-packages-pr-body.md"
cat >"${PR_BODY_FILE}" <<EOF
## Summary
- Update qrrun Linux package repositories to ${VERSION}

## Source Release
- https://github.com/${SOURCE_REPO}/releases/tag/${VERSION}

## Tracking
- qrrun issue: ${ISSUE_URL}

## Repositories
- apt base: https://raw.githubusercontent.com/${LINUX_PACKAGES_REPO}/main/apt
- rpm base: https://raw.githubusercontent.com/${LINUX_PACKAGES_REPO}/main/rpm
- key base: https://raw.githubusercontent.com/${LINUX_PACKAGES_REPO}/main/keys
EOF

gh_packages pr create \
  --repo "${LINUX_PACKAGES_REPO}" \
  --base "${DEFAULT_BRANCH}" \
  --head "${PACKAGES_OWNER}:${BRANCH_NAME}" \
  --title "Update qrrun Linux packages to ${VERSION}" \
  --body-file "${PR_BODY_FILE}"

log "Linux packages PR created successfully"
