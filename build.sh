#!/bin/bash

# IP-Updater Build Script for Linux Debian
# This script builds the IP-Updater service for deployment on Linux Debian systems

set -e  # Exit on any error

echo "Starting IP-Updater build process..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build configuration
BINARY_NAME="ip-updater"
BUILD_DIR="build"
VERSION="1.0.0"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

echo -e "${BLUE}Build Configuration:${NC}"
echo "  Binary Name: ${BINARY_NAME}"
echo "  Version: ${VERSION}"
echo "  Build Time: ${BUILD_TIME}"
echo "  Git Commit: ${GIT_COMMIT}"
echo "  Target: Linux AMD64"
echo ""

# Create build directory
echo -e "${YELLOW}Creating build directory...${NC}"
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Download dependencies
echo -e "${YELLOW}Downloading Go modules...${NC}"
go mod download
go mod tidy

# Run tests (optional, uncomment if you have tests)
# echo -e "${YELLOW}Running tests...${NC}"
# go test -v ./...

# Build for Linux AMD64
echo -e "${YELLOW}Building for Linux AMD64...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="${LDFLAGS}" \
    -o ${BUILD_DIR}/${BINARY_NAME} \
    ./cmd/ip-updater

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Build successful!${NC}"
else
    echo -e "${RED}✗ Build failed!${NC}"
    exit 1
fi

# Create systemd service file
echo -e "${YELLOW}Creating systemd service file...${NC}"
cat > ${BUILD_DIR}/ip-updater.service << EOF
[Unit]
Description=IP Updater Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/ip-updater -config=/etc/ip_updater/config.conf
Restart=always
RestartSec=10
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

# Create installation script
echo -e "${YELLOW}Creating installation script...${NC}"
cat > ${BUILD_DIR}/install.sh << 'EOF'
#!/bin/bash

set -e

BINARY_NAME="ip-updater"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ip_updater"
LOG_DIR="/var/log/ip_updater"
SERVICE_FILE="ip-updater.service"

echo "Installing IP-Updater..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Stop service if running
if systemctl is-active --quiet ip-updater; then
    echo "Stopping existing service..."
    systemctl stop ip-updater
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
echo "  sudo systemctl enable ip-updater"
echo "  sudo systemctl start ip-updater"
echo ""
echo "To check service status:"
echo "  sudo systemctl status ip-updater"
echo ""
echo "To view logs:"
echo "  sudo journalctl -u ip-updater -f"
echo ""
echo "Configuration file: ${CONFIG_DIR}/config.conf"
echo "Log file: ${LOG_DIR}/ip_updater.log"
EOF

chmod +x ${BUILD_DIR}/install.sh

# Create uninstall script
echo -e "${YELLOW}Creating uninstall script...${NC}"
cat > ${BUILD_DIR}/uninstall.sh << 'EOF'
#!/bin/bash

set -e

BINARY_NAME="ip-updater"
INSTALL_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/ip-updater.service"

echo "Uninstalling IP-Updater..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Stop and disable service
if systemctl is-active --quiet ip-updater; then
    echo "Stopping service..."
    systemctl stop ip-updater
fi

if systemctl is-enabled --quiet ip-updater; then
    echo "Disabling service..."
    systemctl disable ip-updater
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
EOF

chmod +x ${BUILD_DIR}/uninstall.sh

# Create README for deployment
echo -e "${YELLOW}Creating deployment README...${NC}"
cat > ${BUILD_DIR}/README.md << EOF
# IP-Updater Deployment

## Installation

1. Copy all files from the build directory to your Linux Debian server
2. Run the installation script as root:
   \`\`\`bash
   sudo ./install.sh
   \`\`\`

## Configuration

Edit the configuration file at \`/etc/ip_updater/config.conf\` to:
- Configure DNS providers (uncomment and fill in your credentials)
- Set up file updaters if needed
- Adjust check intervals and retry settings

## Starting the Service

\`\`\`bash
sudo systemctl enable ip-updater
sudo systemctl start ip-updater
\`\`\`

## Monitoring

Check service status:
\`\`\`bash
sudo systemctl status ip-updater
\`\`\`

View logs:
\`\`\`bash
sudo journalctl -u ip-updater -f
\`\`\`

Or check the log file:
\`\`\`bash
sudo tail -f /var/log/ip_updater/ip_updater.log
\`\`\`

## Uninstallation

Run the uninstall script:
\`\`\`bash
sudo ./uninstall.sh
\`\`\`

## Binary Information

- Version: ${VERSION}
- Build Time: ${BUILD_TIME}
- Git Commit: ${GIT_COMMIT}
- Target: Linux AMD64
EOF

# Show build summary
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}         Build Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Build artifacts created in: ${BUILD_DIR}/"
echo "Files:"
echo "  ${BINARY_NAME}           - Main executable"
echo "  ip-updater.service      - Systemd service file"
echo "  install.sh              - Installation script"
echo "  uninstall.sh            - Uninstallation script"
echo "  README.md               - Deployment guide"
echo ""
echo -e "${BLUE}File sizes:${NC}"
ls -lh ${BUILD_DIR}/

echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Copy the build directory to your target server"
echo "2. Run './install.sh' as root on the target server"
echo "3. Configure /etc/ip_updater/config.conf"
echo "4. Start the service: systemctl start ip-updater"
echo ""
echo -e "${GREEN}Happy deploying! 🚀${NC}"