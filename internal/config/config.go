package config

import (
	"ip-updater/internal/crypto"
	"ip-updater/internal/detector"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	CheckInterval int               `toml:"check_interval"`
	IPDetection   detector.Config   `toml:"ip_detection"`
	DNSUpdaters   []DNSUpdater      `toml:"dns_updater"`
	FileUpdaters  []FileUpdater     `toml:"file_updater"`
	Retry         RetryConfig       `toml:"retry"`
	Logging       LoggingConfig     `toml:"logging"`
}

type DNSUpdater struct {
	Name         string            `toml:"name"`
	Provider     string            `toml:"provider"`
	AccessKey    string            `toml:"access_key"`
	SecretKey    string            `toml:"secret_key"`
	Token        string            `toml:"token"`
	Domain       string            `toml:"domain"`
	Records      []DNSRecord       `toml:"record"`
	ExtraConfig  map[string]string `toml:"extra_config"`
}

type DNSRecord struct {
	Name string `toml:"name"`
	Type string `toml:"type"`
	TTL  int    `toml:"ttl"`
}

type FileUpdater struct {
	Name     string `toml:"name"`
	FilePath string `toml:"file_path"`
	Format   string `toml:"format"`
	KeyPath  string `toml:"key_path"`
	Backup   bool   `toml:"backup"`
}

type RetryConfig struct {
	Interval   int `toml:"interval"`
	MaxRetries int `toml:"max_retries"`
}

type LoggingConfig struct {
	Level    string `toml:"level"`
	FilePath string `toml:"file_path"`
	MaxSize  int    `toml:"max_size"`
	MaxAge   int    `toml:"max_age"`
}

func Load(configPath string) (*Config, error) {
	// Create default config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return nil, err
		}
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.CheckInterval == 0 {
		config.CheckInterval = 600 // 10 minutes
	}

	if len(config.IPDetection.APIEndpoints) == 0 {
		config.IPDetection.APIEndpoints = []string{
			"https://api.ipify.org",
			"https://ipv4.icanhazip.com",
			"https://checkip.amazonaws.com",
		}
	}

	if len(config.IPDetection.WebEndpoints) == 0 {
		config.IPDetection.WebEndpoints = []string{
			"https://ifconfig.me/ip",
			"https://ipinfo.io/ip",
		}
	}

	if config.IPDetection.Timeout == 0 {
		config.IPDetection.Timeout = 30
	}

	if config.Retry.Interval == 0 {
		config.Retry.Interval = 60
	}

	if config.Retry.MaxRetries == 0 {
		config.Retry.MaxRetries = -1 // infinite
	}

	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	if config.Logging.FilePath == "" {
		config.Logging.FilePath = "/var/log/ip_updater/ip_updater.log"
	}

	// Decrypt sensitive data
	if err := decryptSensitiveData(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func createDefaultConfig(configPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	defaultConfig := `# IP-Updater Configuration File

# Check interval in seconds (default: 600 = 10 minutes)
check_interval = 600

[ip_detection]
# Timeout for IP detection requests in seconds
timeout = 30

# API endpoints for getting public IP (tried first)
api_endpoints = [
    "https://api.ipify.org",
    "https://ipv4.icanhazip.com",
    "https://checkip.amazonaws.com"
]

# Web endpoints for getting public IP (fallback)
web_endpoints = [
    "https://ifconfig.me/ip",
    "https://ipinfo.io/ip"
]

[retry]
# Retry interval in seconds when update fails
interval = 60
# Maximum retry attempts (-1 for infinite)
max_retries = -1

[logging]
# Log level: debug, info, warn, error
level = "info"
# Log file path
file_path = "/var/log/ip_updater/ip_updater.log"
# Max log file size in MB
max_size = 100
# Max age of log files in days
max_age = 30

# Example DNS updater configurations (uncomment and configure as needed)

# [[dns_updater]]
# name = "aliyun-example"
# provider = "aliyun"
# access_key = "your_access_key_id"        # Will be encrypted
# secret_key = "your_access_key_secret"    # Will be encrypted
# domain = "example.com"
# [[dns_updater.record]]
# name = "www"
# type = "A"
# ttl = 600

# [[dns_updater]]
# name = "tencent-example"
# provider = "tencent"
# access_key = "your_secret_id"            # Will be encrypted
# secret_key = "your_secret_key"           # Will be encrypted
# domain = "example.com"
# [[dns_updater.record]]
# name = "@"
# type = "A"
# ttl = 300

# [[dns_updater]]
# name = "huawei-example"
# provider = "huawei"
# access_key = "your_access_key"           # Will be encrypted
# secret_key = "your_secret_access_key"    # Will be encrypted
# domain = "example.com"
# [[dns_updater.record]]
# name = "subdomain"
# type = "A"
# ttl = 300

# [[dns_updater]]
# name = "cloudflare-example"
# provider = "cloudflare"
# token = "your_api_token"                 # Will be encrypted
# domain = "example.com"
# [[dns_updater.record]]
# name = "api"
# type = "A"
# ttl = 1

# [[dns_updater]]
# name = "godaddy-example"
# provider = "godaddy"
# access_key = "your_api_key"              # Will be encrypted
# secret_key = "your_api_secret"           # Will be encrypted
# domain = "example.com"
# [[dns_updater.record]]
# name = "mail"
# type = "A"
# ttl = 3600

# Example file updater configurations

# [[file_updater]]
# name = "json-config-example"
# file_path = "/etc/myapp/config.json"
# format = "json"
# key_path = "server/public_ip"           # JSON path: server.public_ip
# backup = true

# [[file_updater]]
# name = "yaml-config-example"
# file_path = "/etc/myapp/config.yaml"
# format = "yaml"
# key_path = "network/external_ip"        # YAML path: network.external_ip
# backup = true

# [[file_updater]]
# name = "toml-config-example"
# file_path = "/etc/myapp/config.toml"
# format = "toml"
# key_path = "server/address"             # TOML path: [server] address
# backup = false

# [[file_updater]]
# name = "ini-config-example"
# file_path = "/etc/myapp/config.ini"
# format = "ini"
# key_path = "network/ip"                 # INI path: [network] ip
# backup = true
`

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

func decryptSensitiveData(config *Config) error {
	for i := range config.DNSUpdaters {
		updater := &config.DNSUpdaters[i]

		if updater.AccessKey != "" {
			decrypted, err := crypto.Decrypt(updater.AccessKey)
			if err == nil {
				updater.AccessKey = decrypted
			}
		}

		if updater.SecretKey != "" {
			decrypted, err := crypto.Decrypt(updater.SecretKey)
			if err == nil {
				updater.SecretKey = decrypted
			}
		}

		if updater.Token != "" {
			decrypted, err := crypto.Decrypt(updater.Token)
			if err == nil {
				updater.Token = decrypted
			}
		}
	}

	return nil
}