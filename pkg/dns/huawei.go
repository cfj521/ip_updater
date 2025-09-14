package dns

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type HuaweiDNSProvider struct {
	accessKey string
	secretKey string
	endpoint  string
	client    *http.Client
}

type HuaweiResponse struct {
	ErrorCode string `json:"errorCode"`
	ErrorMsg  string `json:"errorMsg"`
}

type HuaweiRecordSetList struct {
	Recordsets []HuaweiRecordSet `json:"recordsets"`
}

type HuaweiRecordSet struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Records []string `json:"records"`
	TTL     int      `json:"ttl"`
	Status  string   `json:"status"`
}

type HuaweiZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type HuaweiZoneList struct {
	Zones []HuaweiZone `json:"zones"`
}

func NewHuaweiProvider() *HuaweiDNSProvider {
	return &HuaweiDNSProvider{
		endpoint: "https://dns.myhuaweicloud.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *HuaweiDNSProvider) GetProviderName() string {
	return "huawei"
}

func (p *HuaweiDNSProvider) SetCredentials(accessKey, secretKey string) {
	p.accessKey = accessKey
	p.secretKey = secretKey
}

func (p *HuaweiDNSProvider) UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error {
	zoneId, err := p.getZoneId(domain)
	if err != nil {
		return err
	}

	recordsetId, err := p.getRecordsetId(zoneId, recordName, recordType, domain)
	if err != nil {
		return err
	}

	recordData := map[string]interface{}{
		"records": []string{newIP},
		"ttl":     ttl,
	}

	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/v2/zones/%s/recordsets/%s", zoneId, recordsetId)
	_, err = p.makeRequest("PUT", url, string(jsonData))
	return err
}

func (p *HuaweiDNSProvider) getZoneId(domain string) (string, error) {
	url := "/v2/zones"
	body, err := p.makeRequest("GET", url, "")
	if err != nil {
		return "", err
	}

	var zoneList HuaweiZoneList
	if err := json.Unmarshal(body, &zoneList); err != nil {
		return "", fmt.Errorf("failed to parse zones response: %v", err)
	}

	for _, zone := range zoneList.Zones {
		if strings.TrimSuffix(zone.Name, ".") == domain {
			return zone.ID, nil
		}
	}

	return "", fmt.Errorf("zone not found for domain: %s", domain)
}

func (p *HuaweiDNSProvider) getRecordsetId(zoneId, recordName, recordType, domain string) (string, error) {
	fullRecordName := recordName + "." + domain + "."
	if recordName == "@" || recordName == "" {
		fullRecordName = domain + "."
	}

	url := fmt.Sprintf("/v2/zones/%s/recordsets", zoneId)
	body, err := p.makeRequest("GET", url, "")
	if err != nil {
		return "", err
	}

	var recordsetList HuaweiRecordSetList
	if err := json.Unmarshal(body, &recordsetList); err != nil {
		return "", fmt.Errorf("failed to parse recordsets response: %v", err)
	}

	for _, recordset := range recordsetList.Recordsets {
		if recordset.Name == fullRecordName && recordset.Type == recordType {
			return recordset.ID, nil
		}
	}

	return "", ErrRecordNotFound
}

func (p *HuaweiDNSProvider) makeRequest(method, path, body string) ([]byte, error) {
	fullURL := p.endpoint + path

	req, err := http.NewRequest(method, fullURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sdk-Date", timestamp)

	authorization := p.generateAuthorization(method, path, body, timestamp)
	req.Header.Set("Authorization", authorization)

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
		var huaweiResp HuaweiResponse
		if err := json.Unmarshal(respBody, &huaweiResp); err == nil {
			if huaweiResp.ErrorCode != "" {
				return nil, fmt.Errorf("huawei API error: %s - %s", huaweiResp.ErrorCode, huaweiResp.ErrorMsg)
			}
		}
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return respBody, nil
}

func (p *HuaweiDNSProvider) generateAuthorization(method, path, body, timestamp string) string {
	algorithm := "SDK-HMAC-SHA256"

	// Create canonical request
	canonicalRequest := p.createCanonicalRequest(method, path, body, timestamp)

	// Create string to sign
	hashedCanonicalRequest := p.sha256hex(canonicalRequest)
	stringToSign := fmt.Sprintf("%s\n%s\n%s", algorithm, timestamp, hashedCanonicalRequest)

	// Calculate signature
	signature := hex.EncodeToString(p.hmacSha256([]byte(p.secretKey), stringToSign))

	// Create authorization header
	authorization := fmt.Sprintf("%s Access=%s, SignedHeaders=content-type;host;x-sdk-date, Signature=%s",
		algorithm, p.accessKey, signature)

	return authorization
}

func (p *HuaweiDNSProvider) createCanonicalRequest(method, path, body, timestamp string) string {
	canonicalURI := path
	canonicalQueryString := ""

	// Canonical headers (must be sorted)
	host := "dns.myhuaweicloud.com"
	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:%s\nx-sdk-date:%s\n", host, timestamp)
	signedHeaders := "content-type;host;x-sdk-date"

	// Request payload hash
	hashedPayload := p.sha256hex(body)

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, hashedPayload)

	return canonicalRequest
}

func (p *HuaweiDNSProvider) sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (p *HuaweiDNSProvider) hmacSha256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}