package detector

import (
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"time"
)

// ipv4Regex 匹配点分十进制 IPv4，用于从任意响应体（纯文本/JSON/HTML）中提取 IP
var ipv4Regex = regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)

type Config struct {
	APIEndpoints []string `toml:"api_endpoints"`
	WebEndpoints []string `toml:"web_endpoints"`
	Timeout      int      `toml:"timeout"` // seconds
}

type Detector struct {
	config Config
	client *http.Client
}

func New(config Config) *Detector {
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	return &Detector{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (d *Detector) GetPublicIP() (string, error) {
	// Try API endpoints first
	for _, endpoint := range d.config.APIEndpoints {
		if ip, err := d.getIPFromEndpoint(endpoint); err == nil {
			return ip, nil
		}
	}

	// Fall back to web endpoints
	for _, endpoint := range d.config.WebEndpoints {
		if ip, err := d.getIPFromEndpoint(endpoint); err == nil {
			return ip, nil
		}
	}

	return "", errors.New("failed to get public IP from all endpoints")
}

func (d *Detector) getIPFromEndpoint(endpoint string) (string, error) {
	resp, err := d.client.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("non-200 status code")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 从响应体中提取 IP，兼容纯文本、JSON、HTML 等各种格式
	ip := extractIP(string(body))
	if ip == "" {
		return "", errors.New("no valid IP found in response")
	}

	return ip, nil
}

// extractIP 从任意文本中提取第一个合法的 IPv4 地址（按出现顺序，取首个有效的）
func extractIP(body string) string {
	for _, candidate := range ipv4Regex.FindAllString(body, -1) {
		if isValidIP(candidate) {
			return candidate
		}
	}
	return ""
}

// isValidIP 校验是否为合法的 IPv4 地址（含 0-255 取值范围检查）
func isValidIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() != nil
}