# qrrun
Tunnel local code. Run via QR.

## Prerequisites

- `qrrun` uses Cloudflare Quick Tunnels, so `cloudflared` must be installed and available in your PATH. For details, see [Cloudflare's Quick Tunnel documentation](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/).

## Usage

Run with a local file:

```bash
qrrun hello.py
```

Run from stdin (`-`):

```bash
echo "print('hello from stdin')" | qrrun -
```

By default, qrrun generates a QR code for opening and running your script in Pythonista3 via Cloudflare Quick Tunnels, unless you explicitly override `--transport` or `--runtime`.

By default, qrrun exits after successful content delivery and a short quiet period (`500ms`).

You can tune this with `--exit-quiet-period`:

```bash
qrrun --exit-quiet-period 300ms hello.py
```

Keep serving requests until interrupted:

```bash
qrrun --keep-serving hello.py
```

Each received request is logged to the console (method/path/remote address), regardless of HTTP verb.

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
