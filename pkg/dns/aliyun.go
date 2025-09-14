package dns

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type AliyunProvider struct {
	accessKey string
	secretKey string
	endpoint  string
	client    *http.Client
}

type AliyunResponse struct {
	RequestId string                 `json:"RequestId"`
	Code      string                 `json:"Code"`
	Message   string                 `json:"Message"`
	Data      map[string]interface{} `json:"Data"`
}

func NewAliyunProvider() *AliyunProvider {
	return &AliyunProvider{
		endpoint: "https://alidns.aliyuncs.com",
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

func (p *AliyunProvider) GetRecord(domain, recordName, recordType string) (string, error) {
	// For now, return an error to indicate that record retrieval is not implemented
	// This allows the update to proceed without comparison
	return "", fmt.Errorf("GetRecord not implemented for Aliyun provider")
}

func (p *AliyunProvider) UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error {
	// First, get the record ID
	recordId, err := p.getRecordId(domain, recordName, recordType)
	if err != nil {
		return err
	}

	// Update the record
	params := map[string]string{
		"Action":        "UpdateDomainRecord",
		"RecordId":      recordId,
		"RR":            recordName,
		"Type":          recordType,
		"Value":         newIP,
		"TTL":           fmt.Sprintf("%d", ttl),
		"Format":        "JSON",
		"Version":       "2015-01-09",
		"AccessKeyId":   p.accessKey,
		"SignatureMethod": "HMAC-SHA1",
		"Timestamp":     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce": fmt.Sprintf("%d", time.Now().UnixNano()),
	}

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
	params := map[string]string{
		"Action":        "DescribeDomainRecords",
		"DomainName":    domain,
		"RRKeyWord":     recordName,
		"Type":          recordType,
		"Format":        "JSON",
		"Version":       "2015-01-09",
		"AccessKeyId":   p.accessKey,
		"SignatureMethod": "HMAC-SHA1",
		"Timestamp":     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce": fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	signature := p.generateSignature("POST", params)
	params["Signature"] = signature

	resp, err := p.makeRequest("POST", params)
	if err != nil {
		return "", err
	}

	if resp.Code != "" && resp.Code != "Success" {
		return "", fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	// Extract record ID from response
	domainRecords, ok := resp.Data["DomainRecords"].(map[string]interface{})
	if !ok {
		return "", ErrRecordNotFound
	}

	records, ok := domainRecords["Record"].([]interface{})
	if !ok || len(records) == 0 {
		return "", ErrRecordNotFound
	}

	record, ok := records[0].(map[string]interface{})
	if !ok {
		return "", ErrRecordNotFound
	}

	recordId, ok := record["RecordId"].(string)
	if !ok {
		return "", ErrRecordNotFound
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

	var aliyunResp AliyunResponse
	if err := json.Unmarshal(body, &aliyunResp); err != nil {
		return nil, err
	}

	return &aliyunResp, nil
}