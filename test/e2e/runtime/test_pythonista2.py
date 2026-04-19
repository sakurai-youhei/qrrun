from urllib.parse import unquote
from textwrap import dedent

from test.e2e.base import E2EPrintURLBase


class TestPythonista2(E2EPrintURLBase):
    def runtime(self) -> str:
        return "pythonista2"

    def transport(self) -> str:
        return "cloudflared"

    def script(self) -> bytes:
        return dedent(f"""
            #!/usr/bin/env python2
            import sys
            sys.stdout.write({self.script_output!r})
        """).encode("utf-8")

    def runner_preamble(self) -> bytes:
        return b""

    def extract_runner(self, url_line: str) -> str:
        prefix, _, code = url_line.partition("pythonista2://?exec=")
        assert not prefix, f"unexpected url line: {url_line}"
        return unquote(code)

    def mock_runtime(self) -> str:
        return "python2"

    def mock_runtime_opts(self) -> list[str]:
        return ["-"]


del E2EPrintURLBase  # avoid accidentally importing this test base class
