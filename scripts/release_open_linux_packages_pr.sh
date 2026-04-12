#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:?VERSION is required}"
LINUX_PACKAGES_PAT="${LINUX_PACKAGES_PAT:?LINUX_PACKAGES_PAT is required}"
QRRUN_GITHUB_TOKEN="${QRRUN_GITHUB_TOKEN:?QRRUN_GITHUB_TOKEN is required}"
SOURCE_REPO="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"

SOURCE_OWNER="${SOURCE_REPO%%/*}"
LINUX_PACKAGES_REPO="${LINUX_PACKAGES_REPO:-${SOURCE_OWNER}/linux-packages}"
PACKAGES_OWNER="${LINUX_PACKAGES_REPO%%/*}"

VERSION_NO_V="${VERSION#v}"
if [[ "${VERSION_NO_V}" =~ -(alpha|beta|rc)(\.|$) ]]; then
  echo "Skipping linux-packages PR for pre-release tag: ${VERSION}"
  exit 0
fi

for tool in gh git curl dpkg-scanpackages apt-ftparchive createrepo_c; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "Required tool is not installed: ${tool}"
    exit 1
  fi
done

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

BASE_RELEASE_URL="https://github.com/${SOURCE_REPO}/releases/download/${VERSION}"
DEB_AMD64="qrrun_${VERSION_NO_V}_amd64.deb"
DEB_ARM64="qrrun_${VERSION_NO_V}_arm64.deb"
RPM_X86_64="qrrun_${VERSION_NO_V}_x86_64.rpm"
RPM_AARCH64="qrrun_${VERSION_NO_V}_aarch64.rpm"

echo "Downloading Linux package assets for ${VERSION}"
curl -fsSL -o "${TMP_DIR}/${DEB_AMD64}" "${BASE_RELEASE_URL}/${DEB_AMD64}"
curl -fsSL -o "${TMP_DIR}/${DEB_ARM64}" "${BASE_RELEASE_URL}/${DEB_ARM64}"
curl -fsSL -o "${TMP_DIR}/${RPM_X86_64}" "${BASE_RELEASE_URL}/${RPM_X86_64}"
curl -fsSL -o "${TMP_DIR}/${RPM_AARCH64}" "${BASE_RELEASE_URL}/${RPM_AARCH64}"

export GH_TOKEN="${LINUX_PACKAGES_PAT}"
if ! gh repo view "${LINUX_PACKAGES_REPO}" >/dev/null 2>&1; then
  echo "::warning::Linux package repository is not accessible: ${LINUX_PACKAGES_REPO}. Skipping automation."
  exit 0
fi

DEFAULT_BRANCH="$(gh repo view "${LINUX_PACKAGES_REPO}" --json defaultBranchRef --jq '.defaultBranchRef.name')"
if [[ -z "${DEFAULT_BRANCH}" || "${DEFAULT_BRANCH}" == "null" ]]; then
  DEFAULT_BRANCH="main"
fi

echo "Cloning ${LINUX_PACKAGES_REPO}"
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

mkdir -p "${APT_POOL_DIR}" "${APT_DIST_DIR}/binary-amd64" "${APT_DIST_DIR}/binary-arm64"
mkdir -p "${RPM_X86_DIR}" "${RPM_AARCH64_DIR}"

cp "${TMP_DIR}/${DEB_AMD64}" "${APT_POOL_DIR}/${DEB_AMD64}"
cp "${TMP_DIR}/${DEB_ARM64}" "${APT_POOL_DIR}/${DEB_ARM64}"
cp "${TMP_DIR}/${RPM_X86_64}" "${RPM_X86_DIR}/${RPM_X86_64}"
cp "${TMP_DIR}/${RPM_AARCH64}" "${RPM_AARCH64_DIR}/${RPM_AARCH64}"

pushd apt >/dev/null
rm -f dists/stable/main/binary-amd64/Packages dists/stable/main/binary-amd64/Packages.gz
rm -f dists/stable/main/binary-arm64/Packages dists/stable/main/binary-arm64/Packages.gz

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
popd >/dev/null

createrepo_c --update --simple-md-filenames "${RPM_X86_DIR}"
createrepo_c --update --simple-md-filenames "${RPM_AARCH64_DIR}"

git add -A apt rpm

if git diff --cached --quiet; then
  echo "No repository changes detected. Skipping commit and PR creation."
  exit 0
fi

git commit -m "Update qrrun Linux packages to ${VERSION}"
git push --set-upstream origin "${BRANCH_NAME}"

ISSUE_TITLE="Release: update Linux package repositories for qrrun ${VERSION_NO_V}"
EXISTING_ISSUE_NUMBER="$(GH_TOKEN="${QRRUN_GITHUB_TOKEN}" gh issue list --repo "${SOURCE_REPO}" --state open --search "\"${ISSUE_TITLE}\" in:title" --json number --jq '.[0].number')"

if [[ -n "${EXISTING_ISSUE_NUMBER}" && "${EXISTING_ISSUE_NUMBER}" != "null" ]]; then
  ISSUE_URL="$(GH_TOKEN="${QRRUN_GITHUB_TOKEN}" gh issue view "${EXISTING_ISSUE_NUMBER}" --repo "${SOURCE_REPO}" --json url --jq '.url')"
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

  ISSUE_URL="$(GH_TOKEN="${QRRUN_GITHUB_TOKEN}" gh issue create --repo "${SOURCE_REPO}" --title "${ISSUE_TITLE}" --body-file "${ISSUE_BODY_FILE}")"
fi

EXISTING_PR_NUMBER="$(gh pr list --repo "${LINUX_PACKAGES_REPO}" --head "${PACKAGES_OWNER}:${BRANCH_NAME}" --state open --json number --jq '.[0].number')"
if [[ -n "${EXISTING_PR_NUMBER}" && "${EXISTING_PR_NUMBER}" != "null" ]]; then
  gh pr comment --repo "${LINUX_PACKAGES_REPO}" "${EXISTING_PR_NUMBER}" --body "qrrun tracking issue: ${ISSUE_URL}"
  echo "Linux packages PR already exists: #${EXISTING_PR_NUMBER}"
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
EOF

gh pr create \
  --repo "${LINUX_PACKAGES_REPO}" \
  --base "${DEFAULT_BRANCH}" \
  --head "${PACKAGES_OWNER}:${BRANCH_NAME}" \
  --title "Update qrrun Linux packages to ${VERSION}" \
  --body-file "${PR_BODY_FILE}"
