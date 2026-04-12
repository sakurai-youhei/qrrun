# Installation

## Linux

```bash
curl -fsSL https://raw.githubusercontent.com/sakurai-youhei/qrrun/main/scripts/install.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/sakurai-youhei/qrrun/main/scripts/install.sh | bash -s -- v0.1.0
```

## macOS (Homebrew)

Install the latest release:

```bash
brew install --formula https://github.com/sakurai-youhei/qrrun/releases/latest/download/qrrun.rb
```

Install a specific version:

```bash
brew install --formula https://github.com/sakurai-youhei/qrrun/releases/download/v0.1.3/qrrun.rb
```

## Windows

```cmd
winget install --id sakurai-youhei.qrrun --source winget
```

Install a specific version:

```cmd
winget install --id sakurai-youhei.qrrun --version 0.1.3 --source winget
```

Install system-wide:

```cmd
winget install --id sakurai-youhei.qrrun --scope machine --source winget
```
