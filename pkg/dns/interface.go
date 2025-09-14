package dns

import "ip-updater/internal/config"

type Provider interface {
	UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error
	GetRecord(domain, recordName, recordType string) (string, error)
	GetProviderName() string
	SetCredentials(accessKey, secretKey string)
}

type DNSManager struct {
	providers map[string]Provider
}

func NewDNSManager() *DNSManager {
	return &DNSManager{
		providers: make(map[string]Provider),
	}
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
		return ErrProviderNotFound
	}

	for _, record := range updater.Records {
		// Get current record value for comparison
		currentIP, err := provider.GetRecord(updater.Domain, record.Name, record.Type)
		if err == nil && currentIP == ip {
			// Current value matches new value, skip update
			continue
		}

		if err := provider.UpdateRecord(updater.Domain, record.Name, record.Type, ip, record.TTL); err != nil {
			return err
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