from urllib.parse import unquote
from textwrap import dedent

from test.e2e.base import E2EPrintURLBase


class TestPythonista3(E2EPrintURLBase):
    def runtime(self) -> str:
        return "pythonista3"

    def transport(self) -> str:
        return "cloudflared"

    def script(self) -> bytes:
        return dedent(
            f"""
            #!/usr/bin/env python3
            import sys
            sys.stdout.buffer.write({self.script_output!r})
            for arg in sys.argv[1:]:
                sys.stdout.buffer.write(b"\\nARG:")
                sys.stdout.buffer.write(arg.encode("utf-8"))
            """
        ).encode("utf-8")

    def runner_preamble(self) -> bytes:
        return dedent(
            """
            import builtins
            import sys

            def exec(source, *args, **kwargs):
                sys.stderr.buffer.write(source.encode("utf-8"))
                return builtins.exec(source, *args, **kwargs)
            """
        ).encode("utf-8")

    def extract_runner(self, url_line: str) -> str:
        prefix, _, code = url_line.partition("pythonista3://?exec=")
        assert not prefix, f"unexpected url line: {url_line}"
        return unquote(code)

    def mock_runtime(self) -> str:
        return "python3"

    def mock_runtime_opts(self) -> list[str]:
        return ["-"]


del E2EPrintURLBase  # avoid accidentally importing this test base class
