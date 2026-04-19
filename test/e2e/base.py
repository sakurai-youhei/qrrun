import os
import shutil
import subprocess
import time
from abc import ABC, abstractmethod
from collections.abc import Sequence
from functools import cached_property
from pathlib import Path
from queue import Empty, Queue
from string import printable
from threading import Thread
from unittest import TestCase
from uuid import uuid4


class QRrunPrintURL(subprocess.Popen):
    def __init__(self, transport: str, runtime: str, script: bytes) -> None:
        super().__init__(
            [
                self.__qrrun(),
                "--transport",
                transport,
                "--transport-stderr",
                "--debug",
                "--runtime",
                runtime,
                "--print-url",
                "-",
            ],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        assert self.stdin, "stdin is not available"
        with self.stdin:
            self.stdin.write(script)

    def wait(self, timeout=5.0):
        try:
            super().wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            self.kill()
            super().wait()

    def stdout_readline(self, *, timeout: float) -> bytes:
        line: Queue[bytes] = Queue(maxsize=1)

        def _stdout_readline():
            if self.stdout is None:
                line.put(b"")
            else:
                line.put(self.stdout.readline())

        Thread(target=_stdout_readline, daemon=True).start()

        try:
            return line.get(timeout=timeout)
        except Empty as e:
            raise TimeoutError("timed out waiting for qrrun URL output") from e

    @staticmethod
    def __qrrun() -> Path:
        for candidate in (Path("./qrrun"), Path("./qrrun.exe")):
            if candidate.is_file() and os.access(candidate, os.X_OK):
                return candidate.absolute()
        raise FileNotFoundError("qrrun executable not found")


class E2EPrintURLBase(TestCase, ABC):
    @abstractmethod
    def runtime(self) -> str: ...

    @abstractmethod
    def transport(self) -> str: ...

    @abstractmethod
    def script(self) -> bytes: ...

    """A script passed to QRrun via stdin, which should print the script output
    to stdout when executed by the runner."""

    @abstractmethod
    def runner_preamble(self) -> bytes: ...

    """A hook to prepend into the runner, which outputs the downloaded script
    to stderr before executing it."""

    @abstractmethod
    def extract_runner(self, url_line: str) -> str: ...

    """Extracts the runner command from the given URL line output by qrrun."""

    @abstractmethod
    def mock_runtime(self) -> str: ...

    @abstractmethod
    def mock_runtime_opts(self) -> Sequence[str]: ...

    @cached_property
    def script_output(self) -> bytes:
        return f"こんにちは, QRrun! {uuid4()} {printable}".encode("utf-8")

    def test_validate_url_roundtrip(self) -> None:
        for command in [self.mock_runtime(), "cloudflared"]:
            if shutil.which(command) is None:
                self.skipTest(f"{command!r} is not available")
            try:
                subprocess.run(
                    [command, "--version"],
                    capture_output=True,
                    check=True,
                )
            except (subprocess.CalledProcessError, OSError) as e:
                self.skipTest(f"failed to run {command!r}: {e}")

        with QRrunPrintURL(
            self.transport(),
            self.runtime(),
            self.script(),
        ) as qrrun:
            try:
                url_line = qrrun.stdout_readline(timeout=10)
            except TimeoutError as e:
                qrrun.kill()
                raise TimeoutError(
                    "timed out waiting for qrrun to output a URL: "
                    f"{qrrun.communicate()!r}"
                ) from e

            if not url_line.strip():
                qrrun.kill()
                self.fail(
                    "qrrun does not output a valid URL: "
                    f"{qrrun.communicate()!r}"
                )

            runner = self.extract_runner(url_line.decode("utf-8"))

            time.sleep(5)
            proc = subprocess.run(
                [
                    self.mock_runtime(),
                    *self.mock_runtime_opts(),
                ],
                input=self.runner_preamble() + runner.encode("utf-8"),
                capture_output=True,
            )

        self.assertEqual(
            proc.returncode,
            0,
            f"process exited with non-zero code {proc.returncode}: "
            f"{proc.stdout!r} {proc.stderr!r}",
        )
        self.assertEqual(
            self.script(),
            proc.stderr,
            "script mismatch: "
            f"expected {self.script()!r}, got {proc.stderr!r}",
        )
        self.assertEqual(
            self.script_output,
            proc.stdout,
            "output mismatch: "
            f"expected {self.script_output!r}, got {proc.stdout!r}",
        )
