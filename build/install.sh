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
