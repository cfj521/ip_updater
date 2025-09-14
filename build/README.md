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
sudo systemctl enable ip-updater
sudo systemctl start ip-updater
```

## Monitoring

Check service status:
```bash
sudo systemctl status ip-updater
```

View logs:
```bash
sudo journalctl -u ip-updater -f
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

- Version: 1.0.0
- Build Time: 2025-09-14_11:45:25
- Git Commit: aaa93a7
- Target: Linux AMD64
