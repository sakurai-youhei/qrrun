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
            """
        ).encode("utf-8")

    def runner_preamble(self) -> bytes:
        return dedent(
            """\
            curl() {
                command curl "$@" | tee >(cat 1>&2)
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
