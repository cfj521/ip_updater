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

	// 优化：对同一域名只查询一次DNS记录
	if dm.logger != nil {
		dm.logger.Infof("📡 获取域名 %s 的所有DNS记录...", updater.Domain)
	}

	records, err := provider.GetRecords(updater.Domain)
	var recordsMap map[string]map[string]string // key: "type", value: map[name->value]

	if err != nil {
		if dm.logger != nil {
			dm.logger.Warnf("⚠️ 无法获取DNS记录列表 %s: %v", updater.Domain, err)
			dm.logger.Infof("🔄 将对所有记录尝试直接更新...")
		}
		recordsMap = make(map[string]map[string]string) // 空映射，所有记录都将被视为新记录
	} else {
		if dm.logger != nil {
			dm.logger.Infof("✅ 成功获取到 %d 条DNS记录", len(records))
		}

		// 构建记录映射表，便于快速查找
		// 使用嵌套map: recordsMap[type][name] = value
		recordsMap = make(map[string]map[string]string)
		for _, rec := range records {
			if recordsMap[rec.Type] == nil {
				recordsMap[rec.Type] = make(map[string]string)
			}
			recordsMap[rec.Type][rec.Name] = rec.Value
		}

		// 调试日志：显示获取到的记录
		if dm.logger != nil {
			for recType, typeRecords := range recordsMap {
				for name, value := range typeRecords {
					dm.logger.Debugf("📋 已有记录: type=%s, name=%s, value=%s", recType, name, value)
				}
			}
		}
	}

	// 处理每个配置的记录
	for _, record := range updater.Records {
		recordKey := updater.Domain + "/" + record.Name + "/" + record.Type

		if dm.logger != nil {
			dm.logger.Infof("🔍 处理DNS记录: %s (类型: %s)", recordKey, record.Type)
		}

		// 在已获取的记录中查找匹配项
		// 支持多种格式的名称：原始名称、@表示根域名、带域名的完整名称
		var currentIP string
		var found bool

		typeRecords, typeExists := recordsMap[record.Type]
		if typeExists {
			// 首先尝试直接匹配
			if currentIP, found = typeRecords[record.Name]; found {
				dm.logger.Debugf("✅ 直接匹配到记录: name=%s", record.Name)
			} else if record.Name == "@" {
				// 尝试匹配空名称（阿里云用空字符串表示根域名）
				if currentIP, found = typeRecords[""]; found {
					dm.logger.Debugf("✅ 通过空名称匹配到根域名记录")
				}
			} else {
				// 尝试匹配带域名的完整名称 (如 "www.example.com")
				fullName := record.Name + "." + updater.Domain
				if currentIP, found = typeRecords[fullName]; found {
					dm.logger.Debugf("✅ 通过完整名称匹配到记录: name=%s", fullName)
				}
			}
		}

		if found {
			if dm.logger != nil {
				dm.logger.Infof("✅ 找到现有DNS记录: %s = '%s'", recordKey, currentIP)
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
