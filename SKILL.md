---
name: qrrun
description: "Use when explaining qrrun usage, flags, runtime behavior, and troubleshooting. For installation steps, direct users to INSTALL.md."
---

<!-- cspell:ignore qrrun -->

# qrrun Skill

## Overview
qrrun is a CLI tool that serves a local script through Cloudflare Quick Tunnel and provides a QR code (or runtime URL) so mobile runtimes can execute the script.

## Installation
For installation instructions, see [INSTALL.md](INSTALL.md).

## Prerequisites
- cloudflared must be installed and available on PATH.
- qrrun uses Cloudflare Quick Tunnels (trycloudflare.com).

## Basic Usage
Run with a local file:

```bash
qrrun hello.py arg1 arg2
```

Run from stdin:

```bash
echo 'print("Hello, QRrun!")' | qrrun - arg1 arg2
```

Command format:

```bash
qrrun [flags] <script|-> [args...]
```

## Common Flags
- --runtime string: Target runtime (default: pythonista3)
- --transport string: Tunnel transport (default: cloudflared)
- --transport-opts string: Extra args for the transport command
- --print-url: Print only the runtime URL instead of QR code
- --keep-serving: Keep serving until interrupted
- --quiet-period duration: Quiet period before exit in default mode (default: 500ms)
- --qr-level string: QR error correction level (L, M, Q, H; default: M)
- --transport-stdout: Show transport stdout
- --transport-stderr: Show transport stderr
- --debug: Show debug logs
- --version: Show version
- --help: Show help

## Typical Flow
1. qrrun reads script content from file or stdin.
2. qrrun starts an embedded local web server.
3. qrrun starts cloudflared and gets a public trycloudflare URL.
4. qrrun generates a runtime URL and prints QR code or URL.
5. Mobile runtime fetches script through the tunnel and executes it.

## Runtime and QR Notes
- Supported runtimes: ashell, pythonista2, pythonista3
- Supported QR error correction levels: L, M, Q, H
- Default runtime is pythonista3, which supports arbitrary Python 3 script.
- ashell runtime supports arbitrary bash script.

## Operational Tips
- Use --print-url when validating URL generation in automation or tests.
- Use --transport-stderr and --debug when diagnosing tunnel startup issues.
- Use --keep-serving for multiple scans/requests during manual testing.
