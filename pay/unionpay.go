package pay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

const (
	unionPayVersion      = "5.1.0"
	unionPayEncoding     = "UTF-8"
	unionPaySignMethod   = "01"
	unionPayBizType      = "000201"
	unionPayAccessType   = "0"
	unionPayChannelType  = "07"
	unionPayFrontGateway = "https://gateway.95516.com/gateway/api/frontTransReq.do"
	unionPayBackGateway  = "https://gateway.95516.com/gateway/api/backTransReq.do"
	unionPayAppGateway   = "https://gateway.95516.com/gateway/api/appTransReq.do"
	unionPayQueryGateway = "https://gateway.95516.com/gateway/api/queryTrans.do"
	unionPayTimeLayout   = "20060102150405"
)

type UnionPayConfig struct {
	MerchantID            string
	CertID                string
	PrivateKeyPEM         string
	PrivateKeyPath        string
	PrivateKeyPFXPath     string
	PrivateKeyPFXPassword string
	SignCertPEM           string
	SignCertPath          string
	VerifyCertPEM         string
	VerifyCertPath        string
	VerifyCertDir         string
	Version               string
	Encoding              string
	SignMethod            string
	CurrencyCode          string
	BizType               string
	AccessType            string
	ChannelType           string
	FrontURL              string
	BackURL               string
	FrontGateway          string
	BackGateway           string
	AppGateway            string
	QueryGateway          string
	HTTPClient            *http.Client
}

type UnionPayClient struct {
	config      UnionPayConfig
	privateKey  *rsa.PrivateKey
	certID      string
	verifyCerts map[string]*x509.Certificate
	httpClient  *http.Client
}

type UnionPayTradeRequest struct {
	OrderID      string
	Amount       int64
	TxnTime      string
	FrontURL     string
	BackURL      string
	CurrencyCode string
	BizType      string
	AccessType   string
	ChannelType  string
	OrderDesc    string
	ReqReserved  string
	ExtraParams  map[string]string
}

type UnionPayQueryRequest struct {
	OrderID     string
	TxnTime     string
	BizType     string
	AccessType  string
	ExtraParams map[string]string
}

type UnionPayRefundRequest struct {
	OrderID         string
	OriginalQueryID string
	Amount          int64
	TxnTime         string
	BackURL         string
	CurrencyCode    string
	BizType         string
	AccessType      string
	ReqReserved     string
	ExtraParams     map[string]string
}

type UnionPayForm struct {
	Action string
	Method string
	Fields map[string]string
}

type UnionPayResponse struct {
	Params map[string]string
}

func NewUnionPayClient(cfg UnionPayConfig) (*UnionPayClient, error) {
	cfg.withDefaults()
	if strings.TrimSpace(cfg.MerchantID) == "" {
		return nil, fmt.Errorf("unionpay merchant id is required")
	}

	privateKey, signCert, err := loadUnionPayPrivateKey(cfg)
	if err != nil {
		return nil, err
	}
	certID := strings.TrimSpace(cfg.CertID)
	if certID == "" && signCert != nil {
		certID = signCert.SerialNumber.String()
	}
	if certID == "" {
		return nil, fmt.Errorf("unionpay cert id is required when no signing certificate is configured")
	}

	verifyCerts, err := loadUnionPayVerifyCerts(cfg)
	if err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &UnionPayClient{
		config:      cfg,
		privateKey:  privateKey,
		certID:      certID,
		verifyCerts: verifyCerts,
		httpClient:  httpClient,
	}, nil
}

func (c *UnionPayClient) Config() UnionPayConfig {
	return c.config
}

func (c *UnionPayClient) CertID() string {
	return c.certID
}

func (c *UnionPayClient) BuildFrontPayForm(req UnionPayTradeRequest) (*UnionPayForm, error) {
	params, err := c.consumeParams(req)
	if err != nil {
		return nil, err
	}
	frontURL := fallback(req.FrontURL, c.config.FrontURL)
	if strings.TrimSpace(frontURL) == "" {
		return nil, fmt.Errorf("front url is required for unionpay front pay")
	}
	backURL := fallback(req.BackURL, c.config.BackURL)
	if strings.TrimSpace(backURL) == "" {
		return nil, fmt.Errorf("back url is required for unionpay front pay")
	}
	params["frontUrl"] = frontURL
	params["backUrl"] = backURL

	signed, err := c.SignParams(params)
	if err != nil {
		return nil, err
	}
	return &UnionPayForm{
		Action: c.config.FrontGateway,
		Method: http.MethodPost,
		Fields: signed,
	}, nil
}

func (c *UnionPayClient) AppPay(ctx context.Context, req UnionPayTradeRequest) (*UnionPayResponse, error) {
	params, err := c.consumeParams(req)
	if err != nil {
		return nil, err
	}
	backURL := fallback(req.BackURL, c.config.BackURL)
	if strings.TrimSpace(backURL) == "" {
		return nil, fmt.Errorf("back url is required for unionpay app pay")
	}
	params["backUrl"] = backURL
	params["channelType"] = fallback(req.ChannelType, "08")
	return c.postSigned(ctx, c.config.AppGateway, params)
}

func (c *UnionPayClient) Query(ctx context.Context, req UnionPayQueryRequest) (*UnionPayResponse, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return nil, fmt.Errorf("order id is required")
	}
	if strings.TrimSpace(req.TxnTime) == "" {
		return nil, fmt.Errorf("txn time is required for unionpay query")
	}
	params := c.baseParams()
	params["txnType"] = "00"
	params["txnSubType"] = "00"
	params["bizType"] = fallback(req.BizType, c.config.BizType)
	params["accessType"] = fallback(req.AccessType, c.config.AccessType)
	params["orderId"] = req.OrderID
	params["txnTime"] = req.TxnTime
	mergeParams(params, req.ExtraParams)
	return c.postSigned(ctx, c.config.QueryGateway, params)
}

func (c *UnionPayClient) Refund(ctx context.Context, req UnionPayRefundRequest) (*UnionPayResponse, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return nil, fmt.Errorf("refund order id is required")
	}
	if strings.TrimSpace(req.OriginalQueryID) == "" {
		return nil, fmt.Errorf("original query id is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	backURL := fallback(req.BackURL, c.config.BackURL)
	if strings.TrimSpace(backURL) == "" {
		return nil, fmt.Errorf("back url is required for unionpay refund")
	}

	params := c.baseParams()
	params["txnType"] = "04"
	params["txnSubType"] = "00"
	params["bizType"] = fallback(req.BizType, c.config.BizType)
	params["accessType"] = fallback(req.AccessType, c.config.AccessType)
	params["orderId"] = req.OrderID
	params["txnTime"] = fallback(req.TxnTime, time.Now().Format(unionPayTimeLayout))
	params["txnAmt"] = fmt.Sprintf("%d", req.Amount)
	params["currencyCode"] = fallback(req.CurrencyCode, c.config.CurrencyCode)
	params["origQryId"] = req.OriginalQueryID
	params["backUrl"] = backURL
	if req.ReqReserved != "" {
		params["reqReserved"] = req.ReqReserved
	}
	mergeParams(params, req.ExtraParams)
	return c.postSigned(ctx, c.config.BackGateway, params)
}

func (c *UnionPayClient) SignParams(params map[string]string) (map[string]string, error) {
	if c.privateKey == nil {
		return nil, fmt.Errorf("unionpay private key is not configured")
	}
	signed := cloneStringMap(params)
	signed["version"] = fallback(signed["version"], c.config.Version)
	signed["encoding"] = fallback(signed["encoding"], c.config.Encoding)
	signed["signMethod"] = fallback(signed["signMethod"], c.config.SignMethod)
	signed["merId"] = fallback(signed["merId"], c.config.MerchantID)
	signed["certId"] = fallback(signed["certId"], c.certID)

	content := canonicalUnionPayParams(signed)
	sum := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return nil, fmt.Errorf("sign unionpay params: %w", err)
	}
	signed["signature"] = base64.StdEncoding.EncodeToString(signature)
	return signed, nil
}

func (c *UnionPayClient) VerifyParams(params map[string]string) error {
	signatureText := params["signature"]
	if strings.TrimSpace(signatureText) == "" {
		return fmt.Errorf("unionpay signature is empty")
	}
	cert, err := c.verifyCert(params["certId"])
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(signatureText)
	if err != nil {
		return fmt.Errorf("decode unionpay signature: %w", err)
	}
	content := canonicalUnionPayParams(params)
	sum := sha256.Sum256([]byte(content))
	if err := rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, sum[:], signature); err != nil {
		return fmt.Errorf("verify unionpay signature: %w", err)
	}
	return nil
}

func (c *UnionPayClient) VerifyNotification(values url.Values) (map[string]string, error) {
	params := valuesToMap(values)
	if err := c.VerifyParams(params); err != nil {
		return nil, err
	}
	return params, nil
}

func (f *UnionPayForm) HTML() string {
	method := fallback(f.Method, http.MethodPost)
	var builder strings.Builder
	builder.WriteString(`<form id="unionpay_submit" name="unionpay_submit" action="`)
	builder.WriteString(html.EscapeString(f.Action))
	builder.WriteString(`" method="`)
	builder.WriteString(html.EscapeString(method))
	builder.WriteString(`">`)

	keys := make([]string, 0, len(f.Fields))
	for key := range f.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		builder.WriteString(`<input type="hidden" name="`)
		builder.WriteString(html.EscapeString(key))
		builder.WriteString(`" value="`)
		builder.WriteString(html.EscapeString(f.Fields[key]))
		builder.WriteString(`">`)
	}
	builder.WriteString(`</form><script>document.forms["unionpay_submit"].submit();</script>`)
	return builder.String()
}

func (r *UnionPayResponse) Get(key string) string {
	if r == nil || r.Params == nil {
		return ""
	}
	return r.Params[key]
}

func (r *UnionPayResponse) Success() bool {
	return r.Get("respCode") == "00"
}

func (r *UnionPayResponse) TN() string {
	return r.Get("tn")
}

func (r *UnionPayResponse) QueryID() string {
	return r.Get("queryId")
}

func (c *UnionPayClient) consumeParams(req UnionPayTradeRequest) (map[string]string, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return nil, fmt.Errorf("order id is required")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	params := c.baseParams()
	params["txnType"] = "01"
	params["txnSubType"] = "01"
	params["bizType"] = fallback(req.BizType, c.config.BizType)
	params["accessType"] = fallback(req.AccessType, c.config.AccessType)
	params["channelType"] = fallback(req.ChannelType, c.config.ChannelType)
	params["orderId"] = req.OrderID
	params["txnTime"] = fallback(req.TxnTime, time.Now().Format(unionPayTimeLayout))
	params["txnAmt"] = fmt.Sprintf("%d", req.Amount)
	params["currencyCode"] = fallback(req.CurrencyCode, c.config.CurrencyCode)
	if req.OrderDesc != "" {
		params["orderDesc"] = req.OrderDesc
	}
	if req.ReqReserved != "" {
		params["reqReserved"] = req.ReqReserved
	}
	mergeParams(params, req.ExtraParams)
	return params, nil
}

func (c *UnionPayClient) postSigned(ctx context.Context, endpoint string, params map[string]string) (*UnionPayResponse, error) {
	signed, err := c.SignParams(params)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	values := url.Values{}
	for key, value := range signed {
		if value != "" {
			values.Set(key, value)
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset="+c.config.Encoding)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unionpay http status %d: %s", response.StatusCode, string(body))
	}

	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse unionpay response: %w", err)
	}
	result := valuesToMap(parsed)
	if result["signature"] != "" && len(c.verifyCerts) > 0 {
		if err := c.VerifyParams(result); err != nil {
			return nil, err
		}
	}
	return &UnionPayResponse{Params: result}, nil
}

func (c *UnionPayClient) baseParams() map[string]string {
	return map[string]string{
		"version":    c.config.Version,
		"encoding":   c.config.Encoding,
		"signMethod": c.config.SignMethod,
		"merId":      c.config.MerchantID,
		"certId":     c.certID,
	}
}

func (c *UnionPayClient) verifyCert(certID string) (*x509.Certificate, error) {
	if len(c.verifyCerts) == 0 {
		return nil, fmt.Errorf("unionpay verify certificate is not configured")
	}
	if certID != "" {
		if cert, ok := c.verifyCerts[certID]; ok {
			return cert, nil
		}
		return nil, fmt.Errorf("unionpay verify certificate %s not found", certID)
	}
	if len(c.verifyCerts) == 1 {
		for _, cert := range c.verifyCerts {
			return cert, nil
		}
	}
	return nil, fmt.Errorf("unionpay cert id is required for verification")
}

func (c *UnionPayConfig) withDefaults() {
	c.Version = fallback(c.Version, unionPayVersion)
	c.Encoding = fallback(c.Encoding, unionPayEncoding)
	c.SignMethod = fallback(c.SignMethod, unionPaySignMethod)
	c.CurrencyCode = fallback(c.CurrencyCode, defaultUnionCurrencyCNY)
	c.BizType = fallback(c.BizType, unionPayBizType)
	c.AccessType = fallback(c.AccessType, unionPayAccessType)
	c.ChannelType = fallback(c.ChannelType, unionPayChannelType)
	c.FrontGateway = fallback(c.FrontGateway, unionPayFrontGateway)
	c.BackGateway = fallback(c.BackGateway, unionPayBackGateway)
	c.AppGateway = fallback(c.AppGateway, unionPayAppGateway)
	c.QueryGateway = fallback(c.QueryGateway, unionPayQueryGateway)
}

func loadUnionPayPrivateKey(cfg UnionPayConfig) (*rsa.PrivateKey, *x509.Certificate, error) {
	if cfg.PrivateKeyPFXPath != "" {
		data, err := os.ReadFile(cfg.PrivateKeyPFXPath)
		if err != nil {
			return nil, nil, fmt.Errorf("read unionpay pfx: %w", err)
		}
		key, cert, err := pkcs12.Decode(data, cfg.PrivateKeyPFXPassword)
		if err != nil {
			return nil, nil, fmt.Errorf("decode unionpay pfx: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("unionpay pfx private key is not rsa")
		}
		return rsaKey, cert, nil
	}

	keyPEM := []byte(strings.TrimSpace(cfg.PrivateKeyPEM))
	if len(keyPEM) == 0 && cfg.PrivateKeyPath != "" {
		data, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("read unionpay private key: %w", err)
		}
		keyPEM = data
	}
	if len(strings.TrimSpace(string(keyPEM))) == 0 {
		return nil, nil, fmt.Errorf("unionpay private key pem, path, or pfx path is required")
	}
	privateKey, err := parseRSAPrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, nil, err
	}

	var signCert *x509.Certificate
	certPEM := []byte(strings.TrimSpace(cfg.SignCertPEM))
	if len(certPEM) == 0 && cfg.SignCertPath != "" {
		data, err := os.ReadFile(cfg.SignCertPath)
		if err != nil {
			return nil, nil, fmt.Errorf("read unionpay sign cert: %w", err)
		}
		certPEM = data
	}
	if len(strings.TrimSpace(string(certPEM))) > 0 {
		certs, err := parseCertificates(certPEM)
		if err != nil {
			return nil, nil, fmt.Errorf("parse unionpay sign cert: %w", err)
		}
		if len(certs) > 0 {
			signCert = certs[0]
		}
	}
	return privateKey, signCert, nil
}

func loadUnionPayVerifyCerts(cfg UnionPayConfig) (map[string]*x509.Certificate, error) {
	certs := make(map[string]*x509.Certificate)
	addCerts := func(data []byte) error {
		parsed, err := parseCertificates(data)
		if err != nil {
			return err
		}
		for _, cert := range parsed {
			if _, ok := cert.PublicKey.(*rsa.PublicKey); ok {
				certs[cert.SerialNumber.String()] = cert
			}
		}
		return nil
	}

	if strings.TrimSpace(cfg.VerifyCertPEM) != "" {
		if err := addCerts([]byte(cfg.VerifyCertPEM)); err != nil {
			return nil, fmt.Errorf("parse unionpay verify cert pem: %w", err)
		}
	}
	if cfg.VerifyCertPath != "" {
		data, err := os.ReadFile(cfg.VerifyCertPath)
		if err != nil {
			return nil, fmt.Errorf("read unionpay verify cert: %w", err)
		}
		if err := addCerts(data); err != nil {
			return nil, fmt.Errorf("parse unionpay verify cert: %w", err)
		}
	}
	if cfg.VerifyCertDir != "" {
		entries, err := os.ReadDir(cfg.VerifyCertDir)
		if err != nil {
			return nil, fmt.Errorf("read unionpay verify cert dir: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".cer" && ext != ".crt" && ext != ".pem" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(cfg.VerifyCertDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("read unionpay verify cert %s: %w", entry.Name(), err)
			}
			if err := addCerts(data); err != nil {
				return nil, fmt.Errorf("parse unionpay verify cert %s: %w", entry.Name(), err)
			}
		}
	}
	return certs, nil
}

func parseRSAPrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	rest := data
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		switch block.Type {
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse pkcs1 private key: %w", err)
			}
			return key, nil
		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse pkcs8 private key: %w", err)
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("private key is not rsa")
			}
			return rsaKey, nil
		}
	}
	return nil, fmt.Errorf("rsa private key pem block not found")
}

func parseCertificates(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	rest := data
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		if strings.Contains(block.Type, "CERTIFICATE") {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			certs = append(certs, cert)
		}
	}
	if len(certs) > 0 {
		return certs, nil
	}
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, err
	}
	return []*x509.Certificate{cert}, nil
}

func canonicalUnionPayParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "signature" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func valuesToMap(values url.Values) map[string]string {
	params := make(map[string]string, len(values))
	for key, value := range values {
		if len(value) > 0 {
			params[key] = value[0]
		}
	}
	return params
}

func cloneStringMap(params map[string]string) map[string]string {
	cloned := make(map[string]string, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func mergeParams(target map[string]string, extra map[string]string) {
	for key, value := range extra {
		if key == "signature" {
			continue
		}
		target[key] = value
	}
}
