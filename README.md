# qrrun
[![CI](https://github.com/sakurai-youhei/qrrun/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/sakurai-youhei/qrrun/actions/workflows/ci.yml)
[![Release](https://github.com/sakurai-youhei/qrrun/actions/workflows/release.yml/badge.svg)](https://github.com/sakurai-youhei/qrrun/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/sakurai-youhei/qrrun)](https://github.com/sakurai-youhei/qrrun/releases)
[![License](https://img.shields.io/github/license/sakurai-youhei/qrrun)](LICENSE)

Tunnel local code. Run via QR.

## Prerequisites

- `cloudflared` must be installed and available in your PATH.
- `qrrun` uses Cloudflare Quick Tunnels (`trycloudflare.com`).

See [Cloudflare Quick Tunnel documentation](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/) for details.

## Usage

Run with a local file:

```bash
qrrun hello.py
```

Run from stdin (`-`):

```bash
echo "print('hello from stdin')" | qrrun -
```

By default, `qrrun` generates a QR code for opening and running your script in Pythonista3 via Cloudflare Quick Tunnels, unless you explicitly override `--transport` or `--runtime`.
For more options and behavior details, run `qrrun --help`.

## Installation

### Linux / macOS

Install from GitHub Releases using the maintained installer script:

```bash
curl -fsSL https://raw.githubusercontent.com/sakurai-youhei/qrrun/main/scripts/install.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/sakurai-youhei/qrrun/main/scripts/install.sh | bash -s -- v0.1.0
```

### Windows (cmd + curl + msiexec)

Run the Windows installer script directly from the repository:

```cmd
curl -fLO https://raw.githubusercontent.com/sakurai-youhei/qrrun/main/scripts/install.cmd && install.cmd
```

Install a specific version:

```cmd
curl -fLO https://raw.githubusercontent.com/sakurai-youhei/qrrun/main/scripts/install.cmd && install.cmd v0.1.0-beta.1
```

Install system-wide:

```cmd
curl -fLO https://raw.githubusercontent.com/sakurai-youhei/qrrun/main/scripts/install.cmd && install.cmd v0.1.0-beta.1 machine
```

## Development Setup (gvm)

This project uses Go `1.24.13`.

1. Install gvm (Go Version Manager) by following the [official installation guide](https://github.com/moovweb/gvm?tab=readme-ov-file#installing).

2. Install the required Go version and activate it:

```bash
make gvm-setup
```

Or run directly:

```bash
gvm install go1.24.13
gvm use go1.24.13
```

The expected version is also stored in `.gvmrc`.

## Release

For release operations (including beta releases), see [AGENTS.md](AGENTS.md).
