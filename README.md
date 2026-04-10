# qrrun
Tunnel local code. Run via QR.

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
