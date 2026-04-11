#!/usr/bin/env bash
set -euo pipefail

if ! command -v cloudflared >/dev/null 2>&1; then
  echo "cloudflared is required for e2e-url-only" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required for e2e-url-only" >&2
  exit 1
fi

python3 - <<'PY' >/dev/null
import requests  # noqa: F401
PY

expected="$(python3 - <<'PY'
import secrets
print(secrets.token_hex(12))
PY
)"

tmp_dir="$(mktemp -d)"
stdout_file="${tmp_dir}/qrrun.stdout"
stderr_file="${tmp_dir}/qrrun.stderr"
qrrun_pid=""

cleanup() {
  if [[ -n "${qrrun_pid}" ]] && kill -0 "${qrrun_pid}" >/dev/null 2>&1; then
    kill "${qrrun_pid}" >/dev/null 2>&1 || true
    wait "${qrrun_pid}" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

script="print('${expected}')"

printf "%s\n" "${script}" | ./qrrun --transport cloudflared --runtime pythonista3 --url-only - >"${stdout_file}" 2>"${stderr_file}" &
qrrun_pid="$!"

for _ in $(seq 1 300); do
  if [[ -s "${stdout_file}" ]]; then
    break
  fi
  if ! kill -0 "${qrrun_pid}" >/dev/null 2>&1; then
    echo "qrrun exited before producing URL" >&2
    cat "${stderr_file}" >&2 || true
    exit 1
  fi
  sleep 0.1
done

if [[ ! -s "${stdout_file}" ]]; then
  echo "timed out waiting for qrrun URL output" >&2
  cat "${stderr_file}" >&2 || true
  exit 1
fi

url_line="$(head -n 1 "${stdout_file}" | tr -d '\r')"
if [[ "${url_line}" != pythonista3://?exec=* ]]; then
  echo "unexpected qrrun output: ${url_line}" >&2
  cat "${stderr_file}" >&2 || true
  exit 1
fi

python_code="${url_line#pythonista3://?exec=}"
actual="$(python3 -c "${python_code}" | tr -d '\r\n')"

if [[ "${actual}" != "${expected}" ]]; then
  echo "e2e mismatch: expected '${expected}', got '${actual}'" >&2
  cat "${stderr_file}" >&2 || true
  exit 1
fi

wait "${qrrun_pid}"
qrrun_pid=""

echo "e2e-url-only passed"