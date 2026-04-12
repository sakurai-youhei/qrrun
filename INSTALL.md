# Installation

## Linux

Install from package repositories (recommended).

Debian / Ubuntu (APT):

```bash
curl -fsSL https://raw.githubusercontent.com/sakurai-youhei/linux-packages/main/keys/qrrun-packages.asc \
	| gpg --dearmor \
	| sudo tee /usr/share/keyrings/qrrun-archive-keyring.gpg >/dev/null

echo "deb [signed-by=/usr/share/keyrings/qrrun-archive-keyring.gpg] https://raw.githubusercontent.com/sakurai-youhei/linux-packages/main/apt stable main" \
	| sudo tee /etc/apt/sources.list.d/qrrun.list >/dev/null

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
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://raw.githubusercontent.com/sakurai-youhei/linux-packages/main/keys/qrrun-packages.asc
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
brew install --formula https://github.com/sakurai-youhei/qrrun/releases/download/v<version>/qrrun.rb
```

## Windows (winget)

Install with winget (recommended):

```cmd
winget install --id sakurai-youhei.qrrun --source winget
```

Install a specific version:

```cmd
winget install --id sakurai-youhei.qrrun --source winget --version <version>
```

Install system-wide:

```cmd
winget install --id sakurai-youhei.qrrun --source winget --scope machine
```
