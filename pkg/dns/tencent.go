package dns

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TencentDNSProvider struct {
	secretId  string
	secretKey string
	endpoint  string
	client    *http.Client
}

type TencentResponse struct {
	Response struct {
		Error *TencentError `json:"Error"`
		Data  interface{}   `json:"Data"`
	} `json:"Response"`
}

type TencentError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type TencentRecordList struct {
	Response struct {
		RecordList []TencentRecord `json:"RecordList"`
		Error      *TencentError   `json:"Error"`
	} `json:"Response"`
}

type TencentRecord struct {
	RecordId uint64 `json:"RecordId"`
	Name     string `json:"Name"`
	Type     string `json:"Type"`
	Value    string `json:"Value"`
	TTL      uint64 `json:"TTL"`
	Status   string `json:"Status"`
}

func NewTencentProvider() *TencentDNSProvider {
	return &TencentDNSProvider{
		endpoint: "https://dnspod.tencentcloudapi.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *TencentDNSProvider) GetProviderName() string {
	return "tencent"
}

func (p *TencentDNSProvider) SetCredentials(accessKey, secretKey string) {
	p.secretId = accessKey
	p.secretKey = secretKey
}

func (p *TencentDNSProvider) UpdateRecord(domain, recordName, recordType, newIP string, ttl int) error {
	recordId, err := p.getRecordId(domain, recordName, recordType)
	if err != nil {
		return err
	}

	params := map[string]string{
		"Action":     "ModifyRecord",
		"Version":    "2021-03-23",
		"Region":     "ap-beijing",
		"Domain":     domain,
		"RecordId":   strconv.FormatUint(recordId, 10),
		"SubDomain":  recordName,
		"RecordType": recordType,
		"RecordLine": "默认",
		"Value":      newIP,
		"TTL":        strconv.Itoa(ttl),
	}

	_, err = p.makeRequest(params)
	return err
}

func (p *TencentDNSProvider) getRecordId(domain, recordName, recordType string) (uint64, error) {
	params := map[string]string{
		"Action":     "DescribeRecordList",
		"Version":    "2021-03-23",
		"Region":     "ap-beijing",
		"Domain":     domain,
		"Subdomain":  recordName,
		"RecordType": recordType,
	}

	body, err := p.makeRequest(params)
	if err != nil {
		return 0, err
	}

	var recordList TencentRecordList
	if err := json.Unmarshal(body, &recordList); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v", err)
	}

	if recordList.Response.Error != nil {
		return 0, fmt.Errorf("tencent API error: %s - %s", recordList.Response.Error.Code, recordList.Response.Error.Message)
	}

	if len(recordList.Response.RecordList) == 0 {
		return 0, ErrRecordNotFound
	}

	return recordList.Response.RecordList[0].RecordId, nil
}

func (p *TencentDNSProvider) makeRequest(params map[string]string) ([]byte, error) {
	timestamp := time.Now().Unix()

	authorization := p.generateAuthorization(params, timestamp)

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	req, err := http.NewRequest("POST", p.endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-TC-Action", params["Action"])
	req.Header.Set("X-TC-Version", params["Version"])
	req.Header.Set("X-TC-Region", params["Region"])
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tencentResp TencentResponse
	if err := json.Unmarshal(body, &tencentResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if tencentResp.Response.Error != nil {
		return nil, fmt.Errorf("tencent API error: %s - %s", tencentResp.Response.Error.Code, tencentResp.Response.Error.Message)
	}

	return body, nil
}

func (p *TencentDNSProvider) generateAuthorization(params map[string]string, timestamp int64) string {
	// TC3-HMAC-SHA256 algorithm
	algorithm := "TC3-HMAC-SHA256"
	service := "dnspod"
	version := params["Version"]
	action := params["Action"]
	region := params["Region"]

	// Step 1: Create canonical request
	httpMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""

	// Canonical headers
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", "application/x-www-form-urlencoded; charset=utf-8", "dnspod.tencentcloudapi.com")
	signedHeaders := "content-type;host"

	// Request payload
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	payload := values.Encode()
	hashedPayload := p.sha256hex(payload)

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpMethod, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, hashedPayload)

	// Step 2: Create string to sign
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := p.sha256hex(canonicalRequest)
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s", algorithm, timestamp, credentialScope, hashedCanonicalRequest)

	// Step 3: Calculate signature
	secretDate := p.hmacSha256([]byte("TC3"+p.secretKey), date)
	secretService := p.hmacSha256(secretDate, service)
	secretSigning := p.hmacSha256(secretService, "tc3_request")
	signature := hex.EncodeToString(p.hmacSha256(secretSigning, stringToSign))

	// Step 4: Create authorization header
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, p.secretId, credentialScope, signedHeaders, signature)

	return authorization
}

func (p *TencentDNSProvider) sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func (p *TencentDNSProvider) hmacSha256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}