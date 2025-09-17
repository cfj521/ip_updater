package dns

import (
	"ip-updater/internal/config"
)

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type DNSRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

type Provider interface {
	UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error
	GetRecords(domain string) ([]DNSRecord, error)
	GetProviderName() string
	SetCredentials(accessKey, secretKey string)
}

type DNSManager struct {
	providers map[string]Provider
	logger    Logger
}

func NewDNSManager() *DNSManager {
	return &DNSManager{
		providers: make(map[string]Provider),
	}
}

func (dm *DNSManager) SetLogger(logger Logger) {
	dm.logger = logger
}

func (dm *DNSManager) RegisterProvider(name string, provider Provider) {
	dm.providers[name] = provider
}

func (dm *DNSManager) GetProvider(name string) (Provider, bool) {
	provider, exists := dm.providers[name]
	return provider, exists
}

func (dm *DNSManager) UpdateDNSRecord(updater config.DNSUpdater, ip string) error {
	provider, exists := dm.GetProvider(updater.Provider)
	if !exists {
		if dm.logger != nil {
			dm.logger.Errorf("DNS provider '%s' not found", updater.Provider)
		}
		return ErrProviderNotFound
	}

	// Set credentials for the provider before using it
	if updater.Provider == "cloudflare" && updater.Token != "" {
		provider.SetCredentials(updater.Token, "")
	} else {
		provider.SetCredentials(updater.AccessKey, updater.SecretKey)
	}

	if dm.logger != nil {
		dm.logger.Infof("📋 DNS查询开始 - 提供商: %s, 域名: %s", updater.Provider, updater.Domain)
	}

	for _, record := range updater.Records {
		recordKey := updater.Domain + "/" + record.Name + "/" + record.Type

		if dm.logger != nil {
			dm.logger.Infof("🔍 查询DNS记录: %s (类型: %s)", recordKey, record.Type)
		}

		// Get current record value for comparison
		records, err := provider.GetRecords(updater.Domain)
		if err != nil {
			if dm.logger != nil {
				dm.logger.Warnf("⚠️ 无法获取DNS记录列表 %s: %v", updater.Domain, err)
				dm.logger.Infof("🔄 尝试直接更新DNS记录...")
			}
		} else {
			// Find the matching record
			var currentIP string
			var found bool
			for _, rec := range records {
				if rec.Name == record.Name && rec.Type == record.Type {
					currentIP = rec.Value
					found = true
					break
				}
			}

			if found {
				if dm.logger != nil {
					dm.logger.Infof("✅ 获取到当前DNS记录值: %s = '%s'", recordKey, currentIP)
				}

				if currentIP == ip {
					if dm.logger != nil {
						dm.logger.Infof("✔️ DNS记录值未变化，跳过更新: %s = '%s'", recordKey, currentIP)
					}
					continue
				}

				if dm.logger != nil {
					dm.logger.Infof("📝 DNS记录值需要更新: %s 从 '%s' 更新为 '%s'", recordKey, currentIP, ip)
				}
			} else {
				if dm.logger != nil {
					dm.logger.Infof("🆕 未找到现有DNS记录，将创建新记录: %s", recordKey)
				}
			}
		}

		if err := provider.UpdateRecord(updater.Domain, record.Name, record.Type, ip, record.TTL); err != nil {
			if dm.logger != nil {
				dm.logger.Errorf("❌ DNS记录更新失败: %s: %v", recordKey, err)
			}
			return err
		}

		if dm.logger != nil {
			dm.logger.Infof("✅ DNS记录更新成功: %s = '%s' (TTL: %d)", recordKey, ip, record.TTL)
		}
	}

	return nil
}

// Initialize all DNS providers
func (dm *DNSManager) InitializeProviders() {
	dm.RegisterProvider("aliyun", NewAliyunProvider())
	dm.RegisterProvider("tencent", NewTencentProvider())
	dm.RegisterProvider("huawei", NewHuaweiProvider())
	dm.RegisterProvider("cloudflare", NewCloudflareProvider())
	dm.RegisterProvider("godaddy", NewGoDaddyProvider())
}
