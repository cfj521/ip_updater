package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GoDaddyDNSProvider struct {
	apiKey    string
	apiSecret string
	endpoint  string
	client    *http.Client
}

type GoDaddyRecord struct {
	Data string `json:"data"`
	Name string `json:"name"`
	TTL  int    `json:"ttl"`
	Type string `json:"type"`
}

type GoDaddyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Fields  []struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		Path        string `json:"path"`
		PathRelated string `json:"pathRelated"`
	} `json:"fields"`
}

func NewGoDaddyProvider() *GoDaddyDNSProvider {
	return &GoDaddyDNSProvider{
		endpoint: "https://api.godaddy.com/v1",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *GoDaddyDNSProvider) GetProviderName() string {
	return "godaddy"
}

func (p *GoDaddyDNSProvider) SetCredentials(accessKey, secretKey string) {
	p.apiKey = accessKey
	p.apiSecret = secretKey
}

func (p *GoDaddyDNSProvider) GetRecord(domain, recordName, recordType string) (string, error) {
	record, err := p.getRecord(domain, recordName, recordType)
	if err != nil {
		return "", err
	}
	return record.Data, nil
}

func (p *GoDaddyDNSProvider) UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error {
	// GoDaddy uses a different approach - we update all records of the same name/type at once
	records := []GoDaddyRecord{
		{
			Data: newIP,
			Name: recordName,
			TTL:  ttl,
			Type: recordType,
		},
	}

	jsonData, err := json.Marshal(records)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/domains/%s/records/%s/%s", domain, recordType, recordName)
	_, err = p.makeRequest("PUT", url, bytes.NewReader(jsonData))
	return err
}

func (p *GoDaddyDNSProvider) getRecord(domain, recordName, recordType string) (*GoDaddyRecord, error) {
	url := fmt.Sprintf("/domains/%s/records/%s/%s", domain, recordType, recordName)

	body, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var records []GoDaddyRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("failed to parse records response: %v", err)
	}

	if len(records) == 0 {
		return nil, ErrRecordNotFound
	}

	return &records[0], nil
}

func (p *GoDaddyDNSProvider) makeRequest(method, path string, body io.Reader) ([]byte, error) {
	fullURL := p.endpoint + path

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", p.apiKey, p.apiSecret))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// GoDaddy returns different status codes for different operations
	if resp.StatusCode >= 400 {
		var gdError GoDaddyError
		if err := json.Unmarshal(respBody, &gdError); err == nil {
			return nil, p.formatGoDaddyError(gdError)
		}
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (p *GoDaddyDNSProvider) formatGoDaddyError(gdError GoDaddyError) error {
	if gdError.Message != "" {
		if len(gdError.Fields) > 0 {
			fieldMsg := ""
			for _, field := range gdError.Fields {
				fieldMsg += fmt.Sprintf(" [%s: %s]", field.Path, field.Message)
			}
			return fmt.Errorf("godaddy API error: %s (code: %s)%s", gdError.Message, gdError.Code, fieldMsg)
		}
		return fmt.Errorf("godaddy API error: %s (code: %s)", gdError.Message, gdError.Code)
	}
	return fmt.Errorf("godaddy API error: unknown error (code: %s)", gdError.Code)
}