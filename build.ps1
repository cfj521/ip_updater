#requires -Version 5.1
<#
    IP-Updater 构建脚本 (Windows / PowerShell)
    交叉编译 Linux/amd64 二进制，并生成 systemd 服务、安装/卸载脚本与部署说明。
    版本号唯一来源：version.txt（完整 x.y.z，不自增）。
#>

$ErrorActionPreference = 'Stop'

# ---- 构建配置 ----
$BinaryName = 'ip_updater'
$BuildDir   = 'build'
$Version    = (Test-Path version.txt) ? (Get-Content version.txt -Raw).Trim() : '0.0.0'
$BuildTime  = (Get-Date).ToUniversalTime().ToString('yyyy-MM-dd_HH:mm:ss')
$GitCommit  = (git rev-parse --short HEAD 2>$null)
if (-not $GitCommit) { $GitCommit = 'unknown' }

Write-Host "Build Configuration:" -ForegroundColor Blue
Write-Host "  Binary Name: $BinaryName"
Write-Host "  Version:     $Version"
Write-Host "  Build Time:  $BuildTime"
Write-Host "  Git Commit:  $GitCommit"
Write-Host "  Target:      Linux AMD64"
Write-Host ""

# 以 LF 换行写出文本文件（bash 脚本不能带 CR）
function Write-LfFile($Path, $Content) {
    $lf = $Content -replace "`r`n", "`n"
    [System.IO.File]::WriteAllText($Path, $lf, (New-Object System.Text.UTF8Encoding $false))
}

# ---- 准备构建目录 ----
Write-Host "Creating build directory..." -ForegroundColor Yellow
if (Test-Path $BuildDir) { Remove-Item $BuildDir -Recurse -Force }
New-Item -ItemType Directory -Path $BuildDir | Out-Null

# ---- 依赖 ----
Write-Host "Downloading Go modules..." -ForegroundColor Yellow
go mod download
go mod tidy

# ---- 编译 Linux/amd64 ----
Write-Host "Building for Linux AMD64..." -ForegroundColor Yellow
$env:CGO_ENABLED = '0'
$env:GOOS = 'linux'
$env:GOARCH = 'amd64'
go build -ldflags "-s -w -X main.Version=$Version" -o "$BuildDir/$BinaryName" ./cmd/ip_updater
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed!" -ForegroundColor Red; exit 1 }
Write-Host "Build successful!" -ForegroundColor Green

# ---- systemd 服务文件 ----
Write-Host "Creating systemd service file..." -ForegroundColor Yellow
$service = @'
[Unit]
Description=IP Updater Service
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/ip_updater -daemon -config=/etc/ip_updater/config.conf
Restart=always
RestartSec=10

# 超时设置
TimeoutStartSec=60
TimeoutStopSec=30
TimeoutSec=90

# 信号处理优化
KillMode=mixed
KillSignal=SIGTERM
SendSIGKILL=yes

# 服务限制
LimitNOFILE=1024
MemoryMax=100M

[Install]
WantedBy=multi-user.target
'@
Write-LfFile "$BuildDir/ip_updater.service" $service

# ---- 安装脚本 ----
Write-Host "Creating installation script..." -ForegroundColor Yellow
$install = @'
#!/bin/bash

set -e

BINARY_NAME="ip_updater"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ip_updater"
LOG_DIR="/var/log/ip_updater"
SERVICE_FILE="ip_updater.service"

echo "Installing IP-Updater..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Stop service if running
if systemctl is-active --quiet ip_updater; then
    echo "Stopping existing service..."
    systemctl stop ip_updater
fi

# Install binary
echo "Installing binary to ${INSTALL_DIR}..."
cp ${BINARY_NAME} ${INSTALL_DIR}/${BINARY_NAME}
chmod +x ${INSTALL_DIR}/${BINARY_NAME}

# Create directories
echo "Creating directories..."
mkdir -p ${CONFIG_DIR}
mkdir -p ${LOG_DIR}

# Install systemd service
echo "Installing systemd service..."
cp ${SERVICE_FILE} /etc/systemd/system/
systemctl daemon-reload

# Create default config if it doesn't exist
if [ ! -f "${CONFIG_DIR}/config.conf" ]; then
    echo "Creating default configuration..."
    ${INSTALL_DIR}/${BINARY_NAME} -config=${CONFIG_DIR}/config.conf &
    sleep 2
    killall ${BINARY_NAME} 2>/dev/null || true
fi

echo "Installation complete!"
echo ""
echo "To enable and start the service:"
echo "  sudo systemctl enable ip_updater"
echo "  sudo systemctl start ip_updater"
echo ""
echo "To check service status:"
echo "  sudo systemctl status ip_updater"
echo ""
echo "To view logs:"
echo "  sudo journalctl -u ip_updater -f"
echo ""
echo "Configuration file: ${CONFIG_DIR}/config.conf"
echo "Log file: ${LOG_DIR}/ip_updater.log"
'@
Write-LfFile "$BuildDir/install.sh" $install

# ---- 卸载脚本 ----
Write-Host "Creating uninstall script..." -ForegroundColor Yellow
$uninstall = @'
#!/bin/bash

set -e

BINARY_NAME="ip_updater"
INSTALL_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/ip_updater.service"

echo "Uninstalling IP-Updater..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Stop and disable service
if systemctl is-active --quiet ip_updater; then
    echo "Stopping service..."
    systemctl stop ip_updater
fi

if systemctl is-enabled --quiet ip_updater; then
    echo "Disabling service..."
    systemctl disable ip_updater
fi

# Remove service file
if [ -f "${SERVICE_FILE}" ]; then
    echo "Removing systemd service..."
    rm -f ${SERVICE_FILE}
    systemctl daemon-reload
fi

# Remove binary
if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
    echo "Removing binary..."
    rm -f ${INSTALL_DIR}/${BINARY_NAME}
fi

echo "Uninstallation complete!"
echo ""
echo "Note: Configuration and log files were not removed."
echo "To remove them manually:"
echo "  sudo rm -rf /etc/ip_updater"
echo "  sudo rm -rf /var/log/ip_updater"
'@
Write-LfFile "$BuildDir/uninstall.sh" $uninstall

# ---- 部署 README（占位符稍后替换，避免 PowerShell 解析反引号/$）----
Write-Host "Creating deployment README..." -ForegroundColor Yellow
$readme = @'
# IP-Updater Deployment

## Installation

1. Copy all files from the build directory to your Linux Debian server
2. Run the installation script as root:
   ```bash
   sudo ./install.sh
   ```

## Configuration

Edit the configuration file at `/etc/ip_updater/config.conf` to:
- Configure DNS providers (uncomment and fill in your credentials)
- Set up file updaters if needed
- Adjust check intervals and retry settings

## Starting the Service

```bash
sudo systemctl enable ip_updater
sudo systemctl start ip_updater
```

## Monitoring

Check service status:
```bash
sudo systemctl status ip_updater
```

View logs:
```bash
sudo journalctl -u ip_updater -f
```

Or check the log file:
```bash
sudo tail -f /var/log/ip_updater/ip_updater.log
```

## Uninstallation

Run the uninstall script:
```bash
sudo ./uninstall.sh
```

## Binary Information

- Version: __VERSION__
- Build Time: __BUILD_TIME__
- Git Commit: __GIT_COMMIT__
- Target: Linux AMD64
'@
$readme = $readme.Replace('__VERSION__', $Version).Replace('__BUILD_TIME__', $BuildTime).Replace('__GIT_COMMIT__', $GitCommit)
Write-LfFile "$BuildDir/README.md" $readme

# ---- 构建摘要 ----
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "         Build Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Build artifacts created in: $BuildDir/"
Get-ChildItem $BuildDir | Format-Table Name, Length -AutoSize
