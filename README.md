# QRrun

[![CI](https://github.com/sakurai-youhei/qrrun/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/sakurai-youhei/qrrun/actions/workflows/ci.yml)
[![Release](https://github.com/sakurai-youhei/qrrun/actions/workflows/release.yml/badge.svg)](https://github.com/sakurai-youhei/qrrun/actions/workflows/release.yml)
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
echo "print('Hello, QRrun!')" | qrrun - arg1 arg2
```

By default, QRrun generates a QR code for opening and running your script in [Pythonista 3](https://apps.apple.com/app/pythonista-3/id1085978097); use `--runtime` to override this behavior.
For more options and behavior details, run `qrrun --help`.

## Installation

See [INSTALL.md](INSTALL.md).

## Development Setup

This project uses Go `1.24.13`.

1. If gvm is not installed, follow the [gvm official installation guide](https://github.com/moovweb/gvm?tab=readme-ov-file#installing).

2. Install and activate the required Go version:

```bash
gvm install go1.24.13
gvm use go1.24.13 --default
```

3. Install pre-commit hooks (highly recommended):

```bash
pre-commit install
```

4. Run the end-to-end test:

```bash
make test-e2e
```

## Release

For release operations, see [AGENTS.md](AGENTS.md).
