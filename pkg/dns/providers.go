package dns

import (
	"errors"
)

// Factory function to create providers with credentials
func CreateProvider(providerName, accessKey, secretKey, token string) (Provider, error) {
	switch providerName {
	case "aliyun":
		provider := NewAliyunProvider()
		provider.SetCredentials(accessKey, secretKey)
		return provider, nil
	case "tencent":
		provider := NewTencentProvider()
		provider.SetCredentials(accessKey, secretKey)
		return provider, nil
	case "huawei":
		provider := NewHuaweiProvider()
		provider.SetCredentials(accessKey, secretKey)
		return provider, nil
	case "cloudflare":
		provider := NewCloudflareProvider()
		if token != "" {
			provider.SetCredentials(token, "")
		} else {
			provider.SetCredentials(accessKey, secretKey)
		}
		return provider, nil
	case "godaddy":
		provider := NewGoDaddyProvider()
		provider.SetCredentials(accessKey, secretKey)
		return provider, nil
	default:
		return nil, errors.New("unsupported DNS provider: " + providerName)
	}
}