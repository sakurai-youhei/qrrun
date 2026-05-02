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
            for arg in sys.argv[1:]:
                sys.stdout.write("\\nARG:")
                sys.stdout.write(arg)
        """).encode("utf-8")

    def runner_preamble(self) -> bytes:
        return dedent(
            """
            import __builtin__
            import sys

            def hook_compile(compile):
                def hooked(source, filename, mode, *args, **kwargs):
                    if mode == "exec" and isinstance(source, basestring):
                        sys.stderr.write(source)
                    return compile(source, filename, mode, *args, **kwargs)
                return hooked

            __builtin__.compile = hook_compile(__builtin__.compile)
            """
        ).encode("utf-8")

    def extract_runner(self, url_line: str) -> str:
        prefix, _, code = url_line.partition("pythonista2://?exec=")
        assert not prefix, f"unexpected url line: {url_line}"
        return unquote(code)

    def mock_runtime(self) -> str:
        return "pypy2"

    def mock_runtime_opts(self) -> list[str]:
        return [
            "-c",
            "import sys;exec compile(sys.stdin.read(), '<stdin>', 'exec')",
        ]


del E2EPrintURLBase  # avoid accidentally importing this test base class
