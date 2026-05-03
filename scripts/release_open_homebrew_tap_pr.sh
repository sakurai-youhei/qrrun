#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:?VERSION is required}"
HOMEBREW_TAP_PAT="${HOMEBREW_TAP_PAT:?HOMEBREW_TAP_PAT is required}"
QRRUN_GITHUB_TOKEN="${QRRUN_GITHUB_TOKEN:?QRRUN_GITHUB_TOKEN is required}"
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

gh_tap() {
  GH_TOKEN="${HOMEBREW_TAP_PAT}" gh "$@"
}

gh_source() {
  GH_TOKEN="${QRRUN_GITHUB_TOKEN}" gh "$@"
}

write_issue_body() {
  local output_file="$1"
  local pr_url="$2"

  cat >"${output_file}" <<EOF
## Summary
- Track homebrew-tap submission for qrrun ${VERSION_NO_V}

## Release
- qrrun release tag: ${VERSION}
- source repo: https://github.com/${SOURCE_REPO}

## Homebrew Tap
- tap repo: https://github.com/${HOMEBREW_TAP_REPO}
- target branch: ${BRANCH_NAME}
- pull request: ${pr_url}
EOF
}

for tool in gh git curl grep; do
  require_tool "${tool}"
done

SOURCE_OWNER="${SOURCE_REPO%%/*}"
HOMEBREW_TAP_REPO="${HOMEBREW_TAP_REPO:-${SOURCE_OWNER}/homebrew-tap}"
TAP_OWNER="${HOMEBREW_TAP_REPO%%/*}"
FORMULA_PATH="${HOMEBREW_FORMULA_PATH:-Formula/qrrun.rb}"

VERSION_NO_V="${VERSION#v}"
if [[ "${VERSION_NO_V}" =~ -(alpha|beta|rc)(\.|$) ]]; then
  log "Skipping homebrew-tap PR for pre-release tag: ${VERSION}"
  exit 0
fi

if ! gh_tap repo view "${HOMEBREW_TAP_REPO}" >/dev/null 2>&1; then
  echo "::warning::Homebrew tap repository is not accessible: ${HOMEBREW_TAP_REPO}. Skipping automation."
  exit 0
fi

DEFAULT_BRANCH="$(gh_tap repo view "${HOMEBREW_TAP_REPO}" --json defaultBranchRef --jq '.defaultBranchRef.name')"
if [[ -z "${DEFAULT_BRANCH}" || "${DEFAULT_BRANCH}" == "null" ]]; then
  DEFAULT_BRANCH="main"
fi
log "Using Homebrew tap default branch: ${DEFAULT_BRANCH}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

FORMULA_URL="https://github.com/${SOURCE_REPO}/releases/download/${VERSION}/qrrun.rb"
log "Downloading Homebrew formula from ${FORMULA_URL}"
curl -fsSL -o "${TMP_DIR}/qrrun.rb" "${FORMULA_URL}"

if ! grep -q '^class Qrrun < Formula$' "${TMP_DIR}/qrrun.rb"; then
  log "Downloaded formula does not look valid: ${FORMULA_URL}"
  exit 1
fi

log "Cloning ${HOMEBREW_TAP_REPO}"
git clone "https://x-access-token:${HOMEBREW_TAP_PAT}@github.com/${HOMEBREW_TAP_REPO}.git" "${TMP_DIR}/homebrew-tap"

cd "${TMP_DIR}/homebrew-tap"
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

git switch "${DEFAULT_BRANCH}"

BRANCH_NAME="qrrun-${VERSION_NO_V}"
if git ls-remote --exit-code --heads origin "${BRANCH_NAME}" >/dev/null 2>&1; then
  git switch --track "origin/${BRANCH_NAME}"
else
  git switch -c "${BRANCH_NAME}"
fi

mkdir -p "$(dirname "${FORMULA_PATH}")"
cp "${TMP_DIR}/qrrun.rb" "${FORMULA_PATH}"

if git diff --quiet -- "${FORMULA_PATH}"; then
  log "No formula changes detected. Skipping commit and PR creation."
  exit 0
fi

git add "${FORMULA_PATH}"
git commit -m "Update qrrun to ${VERSION}"
git push --set-upstream origin "${BRANCH_NAME}"

ISSUE_TITLE="Release: update Homebrew tap for qrrun ${VERSION_NO_V}"
ISSUE_BODY_FILE="${TMP_DIR}/qrrun-homebrew-release-issue.md"
EXISTING_ISSUE_NUMBER="$(gh_source issue list --repo "${SOURCE_REPO}" --state open --search "\"${ISSUE_TITLE}\" in:title" --json number --jq '.[0].number')"

if [[ -n "${EXISTING_ISSUE_NUMBER}" && "${EXISTING_ISSUE_NUMBER}" != "null" ]]; then
  ISSUE_NUMBER="${EXISTING_ISSUE_NUMBER}"
  ISSUE_URL="$(gh_source issue view "${EXISTING_ISSUE_NUMBER}" --repo "${SOURCE_REPO}" --json url --jq '.url')"
else
  write_issue_body "${ISSUE_BODY_FILE}" "(pending)"
  ISSUE_URL="$(gh_source issue create --repo "${SOURCE_REPO}" --title "${ISSUE_TITLE}" --body-file "${ISSUE_BODY_FILE}")"
  ISSUE_NUMBER="${ISSUE_URL##*/}"
fi

EXISTING_PR_NUMBER="$(gh_tap pr list --repo "${HOMEBREW_TAP_REPO}" --head "${TAP_OWNER}:${BRANCH_NAME}" --state open --json number --jq '.[0].number')"
if [[ -n "${EXISTING_PR_NUMBER}" && "${EXISTING_PR_NUMBER}" != "null" ]]; then
  PR_URL="$(gh_tap pr view --repo "${HOMEBREW_TAP_REPO}" "${EXISTING_PR_NUMBER}" --json url --jq '.url')"
  write_issue_body "${ISSUE_BODY_FILE}" "${PR_URL}"
  gh_source issue edit "${ISSUE_NUMBER}" --repo "${SOURCE_REPO}" --body-file "${ISSUE_BODY_FILE}" >/dev/null
  gh_tap pr comment --repo "${HOMEBREW_TAP_REPO}" "${EXISTING_PR_NUMBER}" --body "qrrun tracking issue: ${ISSUE_URL}"
  log "Homebrew tap PR already exists: #${EXISTING_PR_NUMBER}"
  exit 0
fi

PR_BODY_FILE="${TMP_DIR}/homebrew-pr-body.md"
cat >"${PR_BODY_FILE}" <<EOF
## Summary
- Update qrrun formula to ${VERSION}

## Source Release
- https://github.com/${SOURCE_REPO}/releases/tag/${VERSION}

## Tracking
- qrrun issue: ${ISSUE_URL}
EOF

PR_URL="$(gh_tap pr create \
  --repo "${HOMEBREW_TAP_REPO}" \
  --base "${DEFAULT_BRANCH}" \
  --head "${TAP_OWNER}:${BRANCH_NAME}" \
  --title "Update qrrun to ${VERSION}" \
  --body-file "${PR_BODY_FILE}")"

write_issue_body "${ISSUE_BODY_FILE}" "${PR_URL}"
gh_source issue edit "${ISSUE_NUMBER}" --repo "${SOURCE_REPO}" --body-file "${ISSUE_BODY_FILE}" >/dev/null

log "Homebrew tap PR created successfully: ${PR_URL}"
