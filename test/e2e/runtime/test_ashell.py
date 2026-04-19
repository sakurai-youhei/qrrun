from base64 import b64encode
from textwrap import dedent

from test.e2e.base import E2EPrintURLBase


class TestAShell(E2EPrintURLBase):
    def runtime(self) -> str:
        return "ashell"

    def transport(self) -> str:
        return "cloudflared"

    def script(self) -> bytes:
        return dedent(
            f"""\
            #!/bin/sh
            echo {b64encode(self.script_output).decode("utf-8")} | base64 -d
            for arg in "$@"; do
                printf '\\nARG:%s' "$arg"
            done
            """
        ).encode("utf-8")

    def runner_preamble(self) -> bytes:
        return dedent(
            """\
            curl() {
                _runner_file="$(mktemp)"
                command curl "$@" >"$_runner_file"
                _status=$?
                if [ "$_status" -ne 0 ]; then
                    rm -f "$_runner_file"
                    return "$_status"
                fi
                cat "$_runner_file" >&2
                cat "$_runner_file"
                rm -f "$_runner_file"
            }
            """
        ).encode("utf-8")

    def extract_runner(self, url_line: str) -> str:
        prefix, _, command = url_line.partition("ashell:")
        assert not prefix, f"unexpected url line: {url_line}"
        return command

    def mock_runtime(self) -> str:
        return "sh"

    def mock_runtime_opts(self) -> list[str]:
        return ["-s"]


del E2EPrintURLBase  # avoid accidentally importing this test base class
