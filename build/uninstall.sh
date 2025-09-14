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
