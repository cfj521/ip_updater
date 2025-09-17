package dns

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	aliyunEndpoint         = "https://alidns.aliyuncs.com"
	aliyunAPIVersion       = "2015-01-09"
	aliyunSignatureMethod  = "HMAC-SHA1"
	aliyunSignatureVersion = "1.0"
	defaultPageSize        = "500"
	timeFormat             = "2006-01-02T15:04:05Z"
)

type AliyunProvider struct {
	accessKey string
	secretKey string
	endpoint  string
	client    *http.Client
}

type AliyunResponse struct {
	RequestId     string                 `json:"RequestId"`
	Code          string                 `json:"Code"`
	Message       string                 `json:"Message"`
	TotalCount    int                    `json:"TotalCount"`
	PageSize      int                    `json:"PageSize"`
	PageNumber    int                    `json:"PageNumber"`
	DomainRecords map[string]interface{} `json:"DomainRecords"`
}

func NewAliyunProvider() *AliyunProvider {
	return &AliyunProvider{
		endpoint: aliyunEndpoint,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *AliyunProvider) GetProviderName() string {
	return "aliyun"
}

func (p *AliyunProvider) SetCredentials(accessKey, secretKey string) {
	p.accessKey = accessKey
	p.secretKey = secretKey
}

func (p *AliyunProvider) GetRecords(domain string) ([]DNSRecord, error) {
	if p.accessKey == "" || p.secretKey == "" {
		return nil, fmt.Errorf("阿里云凭证未设置 (AccessKey: %s, SecretKey: %s)",
			maskCredential(p.accessKey), maskCredential(p.secretKey))
	}

	params := p.buildBaseParams()
	params["Action"] = "DescribeDomainRecords"
	params["DomainName"] = domain
	params["PageSize"] = defaultPageSize

	// Debug mode can be enabled by setting environment variable
	// fmt.Printf("📤 API请求参数 (域名: %s, 操作: %s)\n", domain, params["Action"])

	signature := p.generateSignature("GET", params)
	params["Signature"] = signature

	resp, err := p.makeRequest("GET", params)
	if err != nil {
		return nil, err
	}

	// Debug: Uncomment for detailed API response debugging
	// fmt.Printf("🔍 阿里云API响应 (域名: %s, 记录数: %d)\n", domain, resp.TotalCount)

	if resp.Code != "" && resp.Code != "Success" {
		return nil, fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	// Check response structure
	if resp.DomainRecords == nil {
		return []DNSRecord{}, nil // No records found
	}

	// Extract records from response
	recordList, ok := resp.DomainRecords["Record"].([]interface{})
	if !ok {
		return []DNSRecord{}, nil
	}

	var records []DNSRecord
	for _, item := range recordList {
		record, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := record["RR"].(string)
		recordType, _ := record["Type"].(string)
		value, _ := record["Value"].(string)
		ttlFloat, _ := record["TTL"].(float64)
		ttl := int(ttlFloat)

		records = append(records, DNSRecord{
			Name:  name,
			Type:  recordType,
			Value: value,
			TTL:   ttl,
		})
	}

	// Debug: Uncomment to see parsed records
	// fmt.Printf("📋 解析到 %d 条DNS记录\n", len(records))

	return records, nil
}

func (p *AliyunProvider) UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error {
	// First, try to get the record ID
	recordId, err := p.getRecordId(domain, recordName, recordType)
	if err != nil {
		// If record doesn't exist, create it
		if errors.Is(err, ErrRecordNotFound) {
			return p.addRecord(domain, recordName, recordType, newIP, ttl)
		}
		return err
	}

	// Update the existing record
	params := p.buildBaseParams()
	params["Action"] = "UpdateDomainRecord"
	params["RecordId"] = recordId
	params["RR"] = recordName
	params["Type"] = recordType
	params["Value"] = newIP
	params["TTL"] = fmt.Sprintf("%d", ttl)

	signature := p.generateSignature("POST", params)
	params["Signature"] = signature

	resp, err := p.makeRequest("POST", params)
	if err != nil {
		return err
	}

	if resp.Code != "" && resp.Code != "Success" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

func (p *AliyunProvider) getRecordId(domain, recordName, recordType string) (string, error) {
	params := p.buildBaseParams()
	params["Action"] = "DescribeDomainRecords"
	params["DomainName"] = domain
	params["RRKeyWord"] = recordName
	params["Type"] = recordType

	signature := p.generateSignature("GET", params)
	params["Signature"] = signature

	resp, err := p.makeRequest("GET", params)
	if err != nil {
		return "", err
	}

	if resp.Code != "" && resp.Code != "Success" {
		return "", fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	// Extract record ID from response
	if resp.DomainRecords == nil {
		return "", ErrRecordNotFound
	}

	records, ok := resp.DomainRecords["Record"].([]interface{})
	if !ok || len(records) == 0 {
		return "", ErrRecordNotFound
	}

	record, ok := records[0].(map[string]interface{})
	if !ok {
		return "", ErrRecordNotFound
	}

	// RecordId can be string or number, handle both cases
	var recordId string
	if id, ok := record["RecordId"].(string); ok {
		recordId = id
	} else if id, ok := record["RecordId"].(float64); ok {
		recordId = fmt.Sprintf("%.0f", id)
	} else {
		return "", fmt.Errorf("invalid RecordId format")
	}

	return recordId, nil
}

func (p *AliyunProvider) generateSignature(method string, params map[string]string) string {
	// Sort parameters
	var keys []string
	for k := range params {
		if k != "Signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Build query string
	var queryParts []string
	for _, key := range keys {
		queryParts = append(queryParts, url.QueryEscape(key)+"="+url.QueryEscape(params[key]))
	}
	queryString := strings.Join(queryParts, "&")

	// Build string to sign
	stringToSign := method + "&" + url.QueryEscape("/") + "&" + url.QueryEscape(queryString)

	// Calculate signature
	h := hmac.New(sha1.New, []byte(p.secretKey+"&"))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

func (p *AliyunProvider) buildBaseParams() map[string]string {
	return map[string]string{
		"Format":           "JSON",
		"Version":          aliyunAPIVersion,
		"AccessKeyId":      p.accessKey,
		"SignatureMethod":  aliyunSignatureMethod,
		"Timestamp":        time.Now().UTC().Format(timeFormat),
		"SignatureVersion": aliyunSignatureVersion,
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
	}
}

func maskCredential(credential string) string {
	if len(credential) <= 8 {
		if len(credential) < 2 {
			return "***"
		}
		return "***" + credential[len(credential)-2:]
	}
	return credential[:4] + "***" + credential[len(credential)-4:]
}

func (p *AliyunProvider) addRecord(domain, recordName, recordType, value string, ttl int) error {
	params := p.buildBaseParams()
	params["Action"] = "AddDomainRecord"
	params["DomainName"] = domain
	params["RR"] = recordName
	params["Type"] = recordType
	params["Value"] = value
	params["TTL"] = fmt.Sprintf("%d", ttl)

	signature := p.generateSignature("POST", params)
	params["Signature"] = signature

	resp, err := p.makeRequest("POST", params)
	if err != nil {
		return err
	}

	if resp.Code != "" && resp.Code != "Success" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

func (p *AliyunProvider) makeRequest(method string, params map[string]string) (*AliyunResponse, error) {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	var req *http.Request
	var err error

	if method == "POST" {
		req, err = http.NewRequest("POST", p.endpoint, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest("GET", p.endpoint+"?"+values.Encode(), nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Debug: Uncomment for detailed HTTP response debugging
	// fmt.Printf("🌐 HTTP Status: %s, Content-Length: %d\n", resp.Status, len(body))

	var aliyunResp AliyunResponse
	if err := json.Unmarshal(body, &aliyunResp); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}

	return &aliyunResp, nil
}
