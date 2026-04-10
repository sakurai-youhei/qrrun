# qrrun
Tunnel local code. Run via QR.

## Prerequisites

- `cloudflared` must be installed and available in your PATH.
- `qrrun` uses Cloudflare Quick Tunnels:
	https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/

## Usage

Run with a local file:

```bash
qrrun --transport cloudflared --runtime pythonista3 hello.py
```

Run from stdin (`-`):

```bash
echo "print('hello from stdin')" | qrrun --transport cloudflared --runtime pythonista3 -
```

Show version:

```bash
qrrun --version
```

Current release stage uses Go-style semantic versioning with pre-release tags,
starting from `v0.1.0-alpha.0`.

## Development Setup (gvm)

This project uses Go `1.24.13`.

1. Install gvm (Go Version Manager) by following the official guide:

https://github.com/moovweb/gvm?tab=readme-ov-file#installing

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
