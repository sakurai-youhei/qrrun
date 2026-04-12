# Installation

## Linux

Install from package repositories (recommended).

Debian / Ubuntu (APT):

```bash
echo "deb [trusted=yes] https://raw.githubusercontent.com/sakurai-youhei/linux-packages/main/apt stable main" | sudo tee /etc/apt/sources.list.d/qrrun.list >/dev/null
sudo apt-get update
sudo apt-get install -y qrrun
```

RHEL / Fedora / Amazon Linux (YUM/DNF):

```bash
sudo tee /etc/yum.repos.d/qrrun.repo >/dev/null <<'EOF'
[qrrun]
name=qrrun
baseurl=https://raw.githubusercontent.com/sakurai-youhei/linux-packages/main/rpm/$basearch
enabled=1
gpgcheck=0
repo_gpgcheck=0
EOF

sudo dnf install -y qrrun || sudo yum install -y qrrun
```

## macOS (Homebrew)

Install from tap (recommended):

```bash
brew tap sakurai-youhei/tap
brew install sakurai-youhei/tap/qrrun
```

Install the latest release without adding tap:

```bash
brew install --formula https://github.com/sakurai-youhei/qrrun/releases/latest/download/qrrun.rb
```

Install a specific version (direct formula URL):

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
