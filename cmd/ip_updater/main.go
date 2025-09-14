package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ip-updater/internal/config"
	"ip-updater/internal/detector"
	"ip-updater/internal/logger"
	"ip-updater/internal/updater"
)

var (
	configFile = flag.String("config", "/etc/ip_updater/config.conf", "Path to configuration file")
	version    = flag.Bool("version", false, "Show version information")
	daemon     = flag.Bool("daemon", false, "Run as daemon")
)

const Version = "1.0.0"

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("IP-Updater v%s\n", Version)
		return
	}

	// Initialize logger
	log := logger.New()

	// Check if running for the first time
	if isFirstRun() {
		if err := handleFirstRun(); err != nil {
			log.Fatalf("Failed to handle first run: %v", err)
		}
	}

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logger with loaded settings
	if err := log.Configure(cfg.Logging.Level, cfg.Logging.FilePath, cfg.Logging.MaxSize, cfg.Logging.MaxAge); err != nil {
		log.Warnf("Failed to configure logger: %v", err)
	}

	// Initialize IP detector
	ipDetector := detector.New(cfg.IPDetection)

	// Initialize updater
	ipUpdater := updater.New(cfg, log)

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("IP-Updater started")

	// Main loop
	ticker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Second)
	defer ticker.Stop()

	var lastIP string

	for {
		select {
		case <-ticker.C:
			currentIP, err := ipDetector.GetPublicIP()
			if err != nil {
				log.Errorf("Failed to get public IP: %v", err)
				continue
			}

			if currentIP != lastIP {
				log.Infof("IP changed from %s to %s", lastIP, currentIP)

				if err := ipUpdater.UpdateAll(currentIP); err != nil {
					log.Errorf("Failed to update IP: %v", err)
				} else {
					log.Infof("Successfully updated IP to %s", currentIP)
					lastIP = currentIP
				}
			}

		case sig := <-sigChan:
			log.Infof("Received signal %v, shutting down", sig)
			return
		}
	}
}

func isFirstRun() bool {
	_, err := os.Stat("/etc/systemd/system/ip-updater.service")
	return os.IsNotExist(err)
}

func handleFirstRun() error {
	fmt.Println("First run detected. Do you want to create systemd service? (y/n): ")
	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "Y" {
		return createSystemdService()
	}

	return nil
}

func createSystemdService() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=IP Updater Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=%s -config=/etc/ip_updater/config.conf
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`, execPath)

	servicePath := "/etc/systemd/system/ip-updater.service"

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(servicePath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}

	fmt.Println("Systemd service created successfully!")
	fmt.Println("To enable and start the service, run:")
	fmt.Println("  sudo systemctl enable ip-updater")
	fmt.Println("  sudo systemctl start ip-updater")

	return nil
}