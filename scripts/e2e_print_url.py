#!/usr/bin/env python3
import os
import queue
import secrets
import shutil
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from functools import cached_property
from textwrap import dedent
from urllib.parse import unquote


def terminate_proc_gracefully(proc: subprocess.Popen) -> None:
    if proc.poll() is None:
        proc.terminate()
        try:
            proc.wait(timeout=3)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait()


def readline(stream, *, timeout: float) -> str:
    result: queue.Queue[str] = queue.Queue(maxsize=1)
    thread = threading.Thread(
        target=lambda: result.put(stream.readline()), daemon=True
    )
    thread.start()

    try:
        return result.get(timeout=timeout)
    except queue.Empty as exc:
        raise TimeoutError("timed out waiting for qrrun URL output") from exc


class TestE2EPrintURL(unittest.TestCase):
    def setUp(self) -> None:
        self.expected = secrets.token_hex(12)
        self.script = f"print('{self.expected}', end='')\n"

    @cached_property
    def qrrun_bin(self) -> str:
        for candidate in ("./qrrun", "./qrrun.exe"):
            if os.path.isfile(candidate) and os.access(candidate, os.X_OK):
                return candidate
        return ""

    def test_require_cloudflared(self) -> None:
        self.assertIsNotNone(
            shutil.which("cloudflared"),
            "cloudflared is required for test-e2e",
        )

    def test_require_qrrun(self) -> None:
        self.assertTrue(
            self.qrrun_bin,
            "qrrun executable is required for test-e2e",
        )

    def test_require_python3(self) -> None:
        version = subprocess.check_output(
            [sys.executable, "--version"],
            stderr=subprocess.STDOUT,
            text=True,
        ).strip()
        self.assertRegex(
            version,
            r"^Python 3(\.|\s|$)",
            f"python3 is required for test-e2e: got {version!r}",
        )

    def test_validate_url_roundtrip(self) -> None:
        qrrun_stderr = tempfile.TemporaryFile(mode="w+t")
        self.addCleanup(qrrun_stderr.close)

        qrrun = subprocess.Popen(
            [
                self.qrrun_bin,
                "--transport",
                "cloudflared",
                "--transport-stderr",
                "--debug",
                "--runtime",
                "pythonista3",
                "--print-url",
                "-",
            ],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=qrrun_stderr,
            text=True,
        )
        self.addCleanup(terminate_proc_gracefully, qrrun)

        assert qrrun.stdin is not None
        assert qrrun.stdout is not None

        self.addCleanup(qrrun.stdin.close)
        self.addCleanup(qrrun.stdout.close)

        qrrun.stdin.write(self.script)
        qrrun.stdin.close()

        try:
            url_line = readline(qrrun.stdout, timeout=30.0).strip()
        except TimeoutError as exc:
            qrrun_stderr.seek(0)
            stderr_text = qrrun_stderr.read().strip()
            self.fail(f"{exc}; qrrun stderr: {stderr_text!r}")

        if not url_line:
            qrrun_stderr.seek(0)
            stderr_text = qrrun_stderr.read().strip()
            self.fail(
                "qrrun does not output a valid URL; "
                f"qrrun stderr: {stderr_text!r}"
            )

        self.assertRegex(
            url_line,
            r"^pythonista3://\?exec=",
            f"unexpected qrrun output: {url_line}",
        )

        code = unquote(url_line.split("exec=", 1)[1])

        time.sleep(5)

        python = subprocess.run(
            [sys.executable, "-"],
            input=dedent(
                """
                import builtins
                import sys

                def exec(source, *args, **kwargs):
                    print(source, file=sys.stderr, end="")
                    return builtins.exec(source, *args, **kwargs)
                """
            )
            + code,
            text=True,
            capture_output=True,
            check=True,
        )

        actual = python.stdout
        self.assertEqual(
            actual,
            self.expected,
            f"output mismatch: expected {self.expected!r}, got {actual!r}",
        )

        transferred = python.stderr
        self.assertEqual(
            transferred.rstrip("\n"),
            self.script.rstrip("\n"),
            f"script mismatch: expected {self.script!r}, got {transferred!r}",
        )


if __name__ == "__main__":
    unittest.main(verbosity=2)
