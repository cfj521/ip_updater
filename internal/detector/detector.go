package detector

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

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
			return strings.TrimSpace(ip), nil
		}
	}

	// Fall back to web endpoints
	for _, endpoint := range d.config.WebEndpoints {
		if ip, err := d.getIPFromEndpoint(endpoint); err == nil {
			return strings.TrimSpace(ip), nil
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

	// Extract IP from response
	ip := strings.TrimSpace(string(body))

	// Basic IP validation
	if !isValidIP(ip) {
		return "", errors.New("invalid IP format")
	}

	return ip, nil
}

func isValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}

		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}