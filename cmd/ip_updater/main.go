package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
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
	log.Infof("DNS check interval: %d minutes", cfg.DNSCheckInterval/60)
	log.Infof("File check interval: %d minutes", cfg.FileCheckInterval/60)

	// 创建分离的定时器
	dnsTicker := time.NewTicker(time.Duration(cfg.DNSCheckInterval) * time.Second)
	defer dnsTicker.Stop()

	fileTicker := time.NewTicker(time.Duration(cfg.FileCheckInterval) * time.Second)
	defer fileTicker.Stop()

	var dnsLastIP string
	var fileLastIP string

	for {
		select {
		case <-dnsTicker.C:
			currentIP, err := ipDetector.GetPublicIP()
			if err != nil {
				log.Errorf("Failed to get public IP for DNS update: %v", err)
				continue
			}

			if currentIP != dnsLastIP {
				log.Infof("DNS check: IP changed from %s to %s", dnsLastIP, currentIP)

				if err := ipUpdater.UpdateDNS(currentIP); err != nil {
					log.Errorf("Failed to update DNS: %v", err)
				} else {
					log.Infof("Successfully updated DNS to %s", currentIP)
					dnsLastIP = currentIP
				}
			} else {
				log.Debugf("DNS check: IP unchanged (%s)", currentIP)
			}

		case <-fileTicker.C:
			currentIP, err := ipDetector.GetPublicIP()
			if err != nil {
				log.Errorf("Failed to get public IP for file update: %v", err)
				continue
			}

			if currentIP != fileLastIP {
				log.Infof("File check: IP changed from %s to %s", fileLastIP, currentIP)

				if err := ipUpdater.UpdateFiles(currentIP); err != nil {
					log.Errorf("Failed to update files: %v", err)
				} else {
					log.Infof("Successfully updated files to %s", currentIP)
					fileLastIP = currentIP
				}
			} else {
				log.Debugf("File check: IP unchanged (%s)", currentIP)
			}

		case sig := <-sigChan:
			log.Infof("Received signal %v, shutting down", sig)
			return
		}
	}
}

