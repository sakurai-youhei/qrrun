# QRrun

[![CI](https://github.com/sakurai-youhei/qrrun/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/sakurai-youhei/qrrun/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/github/checks-status/sakurai-youhei/qrrun/main?label=CodeQL)](https://github.com/sakurai-youhei/qrrun/security/code-scanning)
[![Latest Release](https://img.shields.io/github/v/release/sakurai-youhei/qrrun)](https://github.com/sakurai-youhei/qrrun/releases)
[![License](https://img.shields.io/github/license/sakurai-youhei/qrrun)](LICENSE)

Tunnel local code. Run via QR.

![QRrun demo](demo.gif)

## Prerequisites

- `cloudflared` must be installed and available in your PATH.
- QRrun uses Cloudflare Quick Tunnels (`trycloudflare.com`). See [Cloudflare Quick Tunnel documentation](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/) for details.

## Usage

Run with a local file:

```bash
qrrun hello.py arg1 arg2
```

Run from stdin (`-`):

```bash
echo 'print("Hello, QRrun!")' | qrrun - arg1 arg2
```

By default, QRrun generates a QR code for opening and running your script in [Pythonista 3](https://apps.apple.com/app/pythonista-3/id1085978097); use `--runtime` to override this behavior.
For more options and behavior details, run `qrrun --help`.

## Installation

See [INSTALL.md](INSTALL.md).

## Development Setup

1. Install [mise](https://mise.jdx.dev/):
2. Trust `mise.toml` (one-time setup):

```bash
mise trust mise.toml
```

3. Install pre-commit hooks (highly recommended):

```bash
pre-commit install
```

4. Run the end-to-end test:

```bash
make test-e2e
```

## Execution Flow

```mermaid
sequenceDiagram
	autonumber
	participant User as User
	participant QRrun as QRrun
	participant Cloudflare as Cloudflare
	participant Camera as Camera App
	participant Pythonista as Pythonista 3

	User->>QRrun: Run qrrun <script|-> [args...]
	QRrun->>QRrun: Start localhost server and register script endpoint
	QRrun->>Cloudflare: Start cloudflared for quick tunnel to localhost server
	Cloudflare-->>QRrun: Return public trycloudflare URL
	QRrun-->>User: Render QR code (runtime URL)
	User->>Camera: Open camera and scan QR code
	Camera-->>Pythonista: Open pythonista3://?exec=...
	Pythonista->>Cloudflare: Request script URL with token
	Cloudflare->>QRrun: Forward HTTPS request to localhost server
	QRrun-->>Cloudflare: Return script content
	Cloudflare-->>Pythonista: Return script payload
	Pythonista->>Pythonista: Execute script with args
	Pythonista-->>User: Display script output
```
