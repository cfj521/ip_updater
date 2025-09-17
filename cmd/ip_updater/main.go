package main

import (
	"context"
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
	"ip-updater/pkg/dns"
)

var (
	configFile = flag.String("config", "/etc/ip_updater/config.conf", "Path to configuration file")
	version    = flag.Bool("version", false, "Show version information")
	daemon     = flag.Bool("daemon", false, "Run as daemon")
	testDNS    = flag.Bool("test-dns", false, "Test DNS provider credentials and connectivity")
)

var Version = "1.1.10" // Will be overridden by build script

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("IP-Updater v%s\n", Version)
		return
	}

	// Initialize logger
	log := logger.New()

	if *testDNS {
		testDNSProviders(*configFile, log)
		return
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

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	log.Infof("IP-Updater v%s started", Version)
	log.Infof("DNS check interval: %d minutes", cfg.DNSCheckInterval/60)
	log.Infof("File check interval: %d minutes", cfg.FileCheckInterval/60)
	log.Infof("Configured DNS updaters: %d", len(cfg.DNSUpdaters))
	log.Infof("Configured file updaters: %d", len(cfg.FileUpdaters))

	// 创建分离的定时器
	dnsTicker := time.NewTicker(time.Duration(cfg.DNSCheckInterval) * time.Second)
	defer dnsTicker.Stop()

	fileTicker := time.NewTicker(time.Duration(cfg.FileCheckInterval) * time.Second)
	defer fileTicker.Stop()

	var dnsLastIP string
	var fileLastIP string

	// Start shutdown handler in separate goroutine
	go func() {
		sig := <-sigChan
		log.Infof("收到信号 %v，开始优雅关闭...", sig)
		cancel() // Cancel context to trigger graceful shutdown
	}()

	// 启动时立即执行一次检测和更新
	log.Info("执行启动时的立即检测...")

	// DNS检测和更新
	currentIP, err := ipDetector.GetPublicIP()
	if err != nil {
		log.ErrorHighlightf("获取公网IP失败(启动检测): %v", err)
	} else {
		log.Infof("当前公网IP: %s", currentIP)

		if len(cfg.DNSUpdaters) > 0 {
			if err := ipUpdater.UpdateDNS(currentIP); err != nil {
				log.ErrorHighlightf("DNS更新失败(启动检测): %v", err)
			} else {
				log.Successf("DNS更新完成(启动检测)，新IP: %s", currentIP)
				dnsLastIP = currentIP
			}
		} else {
			log.Debugf("未配置DNS更新器，跳过DNS更新(启动检测)")
			dnsLastIP = currentIP
		}

		if len(cfg.FileUpdaters) > 0 {
			if err := ipUpdater.UpdateFiles(currentIP); err != nil {
				log.ErrorHighlightf("文件更新失败(启动检测): %v", err)
			} else {
				log.Successf("文件更新完成(启动检测)，新IP: %s", currentIP)
				fileLastIP = currentIP
			}
		} else {
			log.Debugf("未配置文件更新器，跳过文件更新(启动检测)")
			fileLastIP = currentIP
		}
	}

	// 启动强制退出定时器
	forceExitTimer := time.AfterFunc(5*time.Second, func() {
		log.WarnHighlight("优雅关闭超时(5秒)，强制退出")
		os.Exit(0)
	})
	forceExitTimer.Stop() // 先停止，等收到取消信号后再启动

	for {
		select {
		case <-ctx.Done():
			log.Info("收到关闭信号，停止定时器...")
			dnsTicker.Stop()
			fileTicker.Stop()

			// 启动强制退出定时器
			forceExitTimer.Reset(5 * time.Second)

			log.Info("优雅关闭完成")
			return

		case <-dnsTicker.C:
			currentIP, err := ipDetector.GetPublicIP()
			if err != nil {
				log.ErrorHighlightf("获取公网IP失败(DNS检查): %v", err)
				continue
			}

			if currentIP != dnsLastIP {
				log.Infof("DNS check: IP changed from %s to %s", dnsLastIP, currentIP)

				if len(cfg.DNSUpdaters) > 0 {
					if err := ipUpdater.UpdateDNS(currentIP); err != nil {
						log.ErrorHighlightf("DNS更新失败: %v", err)
					} else {
						log.Successf("DNS更新完成，新IP: %s", currentIP)
						dnsLastIP = currentIP
					}
				} else {
					log.Debugf("No DNS updaters configured, skipping DNS update")
					dnsLastIP = currentIP
				}
			} else {
				log.Debugf("DNS check: IP unchanged (%s)", currentIP)
			}

		case <-fileTicker.C:
			currentIP, err := ipDetector.GetPublicIP()
			if err != nil {
				log.ErrorHighlightf("获取公网IP失败(文件检查): %v", err)
				continue
			}

			if currentIP != fileLastIP {
				log.Infof("File check: IP changed from %s to %s", fileLastIP, currentIP)

				if len(cfg.FileUpdaters) > 0 {
					if err := ipUpdater.UpdateFiles(currentIP); err != nil {
						log.ErrorHighlightf("文件更新失败: %v", err)
					} else {
						log.Successf("文件更新完成，新IP: %s", currentIP)
						fileLastIP = currentIP
					}
				} else {
					log.Debugf("No file updaters configured, skipping file update")
					fileLastIP = currentIP
				}
			} else {
				log.Debugf("File check: IP unchanged (%s)", currentIP)
			}

		}
	}
}

func testDNSProviders(configFile string, log *logger.Logger) {
	log.Info("🧪 开始DNS凭证测试...")

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		log.ErrorHighlightf("配置文件加载失败: %v", err)
		os.Exit(1)
	}

	if len(cfg.DNSUpdaters) == 0 {
		log.WarnHighlight("未找到DNS更新器配置")
		os.Exit(1)
	}

	// Initialize DNS manager
	dnsManager := dns.NewDNSManager()
	dnsManager.SetLogger(log)
	dnsManager.InitializeProviders()

	// Test each DNS updater
	for i, updater := range cfg.DNSUpdaters {
		log.Infof("\n📋 测试DNS更新器 #%d: %s", i+1, updater.Name)
		log.Infof("提供商: %s", updater.Provider)
		log.Infof("域名: %s", updater.Domain)

		// Mask credentials for logging
		maskedKey := maskCredential(updater.AccessKey)
		maskedSecret := maskCredential(updater.SecretKey)
		log.Infof("AccessKey: %s", maskedKey)
		log.Infof("SecretKey: %s", maskedSecret)

		// Test connectivity
		testResult := testSingleDNSProvider(dnsManager, updater, log)
		if testResult {
			log.Successf("✅ DNS提供商 %s 测试成功", updater.Name)
		} else {
			log.ErrorHighlightf("❌ DNS提供商 %s 测试失败", updater.Name)
		}
	}

	log.Info("\n🧪 DNS凭证测试完成")
}

func testSingleDNSProvider(dnsManager *dns.DNSManager, updater config.DNSUpdater, log *logger.Logger) bool {
	provider, exists := dnsManager.GetProvider(updater.Provider)
	if !exists {
		log.ErrorHighlightf("不支持的DNS提供商: %s", updater.Provider)
		return false
	}

	// Set credentials
	if updater.Provider == "cloudflare" && updater.Token != "" {
		provider.SetCredentials(updater.Token, "")
	} else {
		provider.SetCredentials(updater.AccessKey, updater.SecretKey)
	}

	log.Infof("🔗 连接测试: 正在验证凭证和记录访问...")

	// Test each configured record directly
	success := true
	log.Infof("\n🔍 开始测试配置的记录:")

	for i, record := range updater.Records {
		log.Infof("   [%d/%d] 测试记录: %s.%s (%s)", i+1, len(updater.Records), record.Name, updater.Domain, record.Type)

		currentValue, err := getRecordFromList(provider, updater.Domain, record.Name, record.Type)
		if err != nil {
			if err.Error() == "DNS record not found" {
				log.Infof("       📝 记录不存在，程序运行时将自动创建")
			} else {
				log.WarnHighlightf("       ⚠️ 记录查询失败: %v", err)
				log.Infof("       💡 可能的原因: API权限不足、域名配置错误或网络问题")
				success = false
			}
		} else {
			log.Successf("       ✅ 记录存在，当前值: %s", currentValue)
		}
	}

	return success
}

func maskCredential(credential string) string {
	if len(credential) <= 8 {
		return "***" + credential[len(credential)-2:]
	}
	return credential[:4] + "***" + credential[len(credential)-4:]
}


// getRecordFromList is a helper function to get a specific record from provider
func getRecordFromList(provider dns.Provider, domain, recordName, recordType string) (string, error) {
	records, err := provider.GetRecords(domain)
	if err != nil {
		return "", err
	}

	for _, rec := range records {
		if rec.Name == recordName && rec.Type == recordType {
			return rec.Value, nil
		}
	}

	return "", fmt.Errorf("DNS record not found")
}
