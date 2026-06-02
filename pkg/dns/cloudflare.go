package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CloudflareDNSProvider struct {
	apiToken string
	endpoint string
	client   *http.Client
}

type CloudflareResponse struct {
	Success bool              `json:"success"`
	Errors  []CloudflareError `json:"errors"`
	Result  interface{}       `json:"result"`
}

type CloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CloudflareRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	ZoneID  string `json:"zone_id"`
}

type CloudflareRecordRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

func NewCloudflareProvider() *CloudflareDNSProvider {
	return &CloudflareDNSProvider{
		endpoint: "https://api.cloudflare.com/client/v4",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *CloudflareDNSProvider) GetRecords(domain string) ([]DNSRecord, error) {
	// 获取 zone_id
	zoneId, err := p.getZoneId(domain)
	if err != nil {
		return nil, fmt.Errorf("获取zone失败: %w", err)
	}

	// 获取所有 DNS 记录
	url := fmt.Sprintf("/zones/%s/dns_records?per_page=100", zoneId)
	body, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response CloudflareResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return nil, p.formatCloudflareErrors(response.Errors)
	}

	records, ok := response.Result.([]interface{})
	if !ok {
		return []DNSRecord{}, nil
	}

	var dnsRecords []DNSRecord
	for _, item := range records {
		recordData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		recordId, _ := recordData["id"].(string)
		recordType, _ := recordData["type"].(string)
		name, _ := recordData["name"].(string)
		content, _ := recordData["content"].(string)
		ttlFloat, _ := recordData["ttl"].(float64)
		ttl := int(ttlFloat)

		// Cloudflare TTL: 1 = 自动，其他为秒数
		if ttl == 1 {
			ttl = 0 // 0 表示自动
		}

		dnsRecords = append(dnsRecords, DNSRecord{
			Name:  name,
			Type:  recordType,
			Value: content,
			TTL:   ttl,
		})

		_ = recordId // 保留供将来使用
	}

	return dnsRecords, nil
}

func (p *CloudflareDNSProvider) GetProviderName() string {
	return "cloudflare"
}

func (p *CloudflareDNSProvider) SetCredentials(accessKey, secretKey string) {
	p.apiToken = accessKey
}

func (p *CloudflareDNSProvider) UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error {
	zoneId, err := p.getZoneId(domain)
	if err != nil {
		return err
	}

	recordId, err := p.getRecordId(zoneId, recordName, recordType, domain)
	if err != nil {
		return err
	}

	recordData := CloudflareRecordRequest{
		Type:    recordType,
		Name:    p.getFullRecordName(recordName, domain),
		Content: newIP,
		TTL:     ttl,
	}

	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/zones/%s/dns_records/%s", zoneId, recordId)
	_, err = p.makeRequest("PUT", url, bytes.NewReader(jsonData))
	return err
}

func (p *CloudflareDNSProvider) getZoneId(domain string) (string, error) {
	url := fmt.Sprintf("/zones?name=%s", domain)
	body, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	var response CloudflareResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse zones response: %v", err)
	}

	if !response.Success {
		return "", p.formatCloudflareErrors(response.Errors)
	}

	zones, ok := response.Result.([]interface{})
	if !ok || len(zones) == 0 {
		return "", fmt.Errorf("zone not found for domain: %s", domain)
	}

	zoneData, ok := zones[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid zone data format")
	}

	zoneId, ok := zoneData["id"].(string)
	if !ok {
		return "", fmt.Errorf("zone ID not found")
	}

	return zoneId, nil
}

func (p *CloudflareDNSProvider) getRecordId(zoneId, recordName, recordType, domain string) (string, error) {
	fullRecordName := p.getFullRecordName(recordName, domain)
	url := fmt.Sprintf("/zones/%s/dns_records?name=%s&type=%s", zoneId, fullRecordName, recordType)

	body, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	var response CloudflareResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse records response: %v", err)
	}

	if !response.Success {
		return "", p.formatCloudflareErrors(response.Errors)
	}

	records, ok := response.Result.([]interface{})
	if !ok || len(records) == 0 {
		return "", ErrRecordNotFound
	}

	recordData, ok := records[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid record data format")
	}

	recordId, ok := recordData["id"].(string)
	if !ok {
		return "", fmt.Errorf("record ID not found")
	}

	return recordId, nil
}

func (p *CloudflareDNSProvider) getFullRecordName(recordName, domain string) string {
	if recordName == "@" || recordName == "" {
		return domain
	}
	return fmt.Sprintf("%s.%s", recordName, domain)
}

func (p *CloudflareDNSProvider) makeRequest(method, path string, body io.Reader) ([]byte, error) {
	fullURL := p.endpoint + path

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var cfResp CloudflareResponse
		if err := json.Unmarshal(respBody, &cfResp); err == nil && !cfResp.Success {
			return nil, p.formatCloudflareErrors(cfResp.Errors)
		}
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return respBody, nil
}

func (p *CloudflareDNSProvider) formatCloudflareErrors(errors []CloudflareError) error {
	if len(errors) == 0 {
		return fmt.Errorf("cloudflare API error: unknown error")
	}

	if len(errors) == 1 {
		return fmt.Errorf("cloudflare API error: %s (code: %d)", errors[0].Message, errors[0].Code)
	}

	var messages []string
	for _, err := range errors {
		messages = append(messages, fmt.Sprintf("%s (code: %d)", err.Message, err.Code))
	}
	return fmt.Errorf("cloudflare API errors: %v", messages)
}
