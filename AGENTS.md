# Agent Instructions for qrrun

This file provides repository-level guidance for any coding agent (Copilot, Claude, Gemini, etc.) operating in this project.

## Release Operations

Use this playbook when asked to create a release (especially beta releases).

### Objective

Create a GitHub release by pushing a version tag that matches `v*`.
This repository uses `.github/workflows/release.yml` for tag-triggered builds and release publishing.

### Preconditions

1. Work from `main`.
2. Ensure local `main` is up to date with `origin/main`.
3. Ensure working tree is clean.
4. Ensure the target tag does not already exist locally or remotely.

### Tag Rules

- Stable example: `v0.2.0`
- Beta example: `v0.2.0-beta.1`
- Tag must start with `v` so the release workflow runs.

### Standard Steps

1. Sync main:

```bash
git switch main
git pull --ff-only
```

2. Verify clean state:

```bash
git status --short --branch
```

3. Verify tag is unused:

```bash
git rev-parse -q --verify refs/tags/<TAG> || true
git ls-remote --tags origin | grep "refs/tags/<TAG>$" || true
```

4. Create signed tag:

```bash
git tag -s <TAG> -m "<TAG>"
```

5. Push tag:

```bash
git push origin <TAG>
```

### Post-Release Verification

1. Confirm workflow started:

```bash
gh run list --workflow release.yml --limit 5
```

2. Confirm release exists after workflow success:

```bash
gh release view <TAG>
```

3. If needed, open the workflow run URL and check failed job logs.

### Failure Handling

- If tag already exists, increment version suffix (for beta, increase `beta.N`).
- If workflow does not trigger, verify tag name starts with `v` and was pushed to `origin`.
- If release job is skipped, check `guard-main` job output in workflow logs.

### Notes

- Do not tag from feature branches.
- Prefer signed tags for release traceability.
- Keep release actions limited to requested scope (do not modify unrelated branches/tags).

## Pull Request Body Editing

Use these rules when creating or editing PR descriptions via `gh pr create` / `gh pr edit`.

### Goal

Avoid malformed Markdown caused by shell escaping or accidental blockquote prefixes.

### Recommended Method

Prefer a body file instead of inline multi-line strings.

```bash
cat > /tmp/pr-body.md <<'EOF'
## Summary
- item 1
- item 2

## Added
- detail A
EOF

gh pr edit <PR_NUMBER> --body-file /tmp/pr-body.md
```

### Rules

- Do not pass literal `\\n` sequences expecting line breaks.
- Do not prefix body lines with `>` unless a blockquote is intended.
- After editing, verify rendered body content:

```bash
gh pr view <PR_NUMBER> --json body
```
