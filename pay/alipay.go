package pay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	alipay "github.com/smartwalle/alipay/v3"
)

type AlipayConfig struct {
	AppID                   string
	PrivateKey              string
	PrivateKeyPath          string
	Production              bool
	NotifyURL               string
	ReturnURL               string
	AppCertPublicKeyPath    string
	AlipayRootCertPath      string
	AlipayCertPublicKeyPath string
	AlipayPublicKey         string
	EncryptKey              string
	SandboxGateway          string
	ProductionGateway       string
	UsePastSandboxGateway   bool
	HTTPClient              *http.Client
}

type AlipayClient struct {
	config AlipayConfig
	client *alipay.Client
}

type AlipayTradeRequest struct {
	Subject        string
	OutTradeNo     string
	Amount         int64
	Body           string
	NotifyURL      string
	ReturnURL      string
	ProductCode    string
	TimeoutExpress string
	TimeExpire     string
	PassbackParams string
	AppAuthToken   string
	QuitURL        string
	QRPayMode      string
	QRCodeWidth    string
}

type AlipayRefundRequest struct {
	OutTradeNo   string
	TradeNo      string
	OutRequestNo string
	Amount       int64
	Reason       string
	AppAuthToken string
	QueryOptions []string
}

func NewAlipayClient(cfg AlipayConfig) (*AlipayClient, error) {
	privateKey, err := cfg.privateKey()
	if err != nil {
		return nil, err
	}

	opts := make([]alipay.OptionFunc, 0, 4)
	if cfg.HTTPClient != nil {
		opts = append(opts, alipay.WithHTTPClient(cfg.HTTPClient))
	}
	if cfg.SandboxGateway != "" {
		opts = append(opts, alipay.WithSandboxGateway(cfg.SandboxGateway))
	}
	if cfg.ProductionGateway != "" {
		opts = append(opts, alipay.WithProductionGateway(cfg.ProductionGateway))
	}
	if cfg.UsePastSandboxGateway {
		opts = append(opts, alipay.WithPastSandboxGateway())
	}

	if strings.TrimSpace(cfg.AppID) == "" {
		return nil, fmt.Errorf("alipay app id is required")
	}
	client, err := alipay.New(cfg.AppID, privateKey, cfg.Production, opts...)
	if err != nil {
		return nil, fmt.Errorf("new alipay client: %w", err)
	}

	if cfg.AppCertPublicKeyPath != "" {
		if err := client.LoadAppCertPublicKeyFromFile(cfg.AppCertPublicKeyPath); err != nil {
			return nil, fmt.Errorf("load alipay app cert public key: %w", err)
		}
	}
	if cfg.AlipayRootCertPath != "" {
		if err := client.LoadAliPayRootCertFromFile(cfg.AlipayRootCertPath); err != nil {
			return nil, fmt.Errorf("load alipay root cert: %w", err)
		}
	}
	if cfg.AlipayCertPublicKeyPath != "" {
		if err := client.LoadAlipayCertPublicKeyFromFile(cfg.AlipayCertPublicKeyPath); err != nil {
			return nil, fmt.Errorf("load alipay cert public key: %w", err)
		}
	}
	if cfg.AlipayPublicKey != "" {
		if err := client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
			return nil, fmt.Errorf("load alipay public key: %w", err)
		}
	}
	if cfg.EncryptKey != "" {
		if err := client.SetEncryptKey(cfg.EncryptKey); err != nil {
			return nil, fmt.Errorf("set alipay encrypt key: %w", err)
		}
	}

	return &AlipayClient{config: cfg, client: client}, nil
}

func (c *AlipayClient) SDKClient() *alipay.Client {
	return c.client
}

func (c *AlipayClient) Config() AlipayConfig {
	return c.config
}

func (c *AlipayClient) PagePay(req AlipayTradeRequest) (*url.URL, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	param := alipay.TradePagePay{
		Trade:       c.trade(req, fallback(req.ProductCode, "FAST_INSTANT_TRADE_PAY")),
		QRPayMode:   req.QRPayMode,
		QRCodeWidth: req.QRCodeWidth,
	}
	return c.client.TradePagePay(param)
}

func (c *AlipayClient) WapPay(req AlipayTradeRequest) (*url.URL, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	param := alipay.TradeWapPay{
		Trade:      c.trade(req, fallback(req.ProductCode, "QUICK_WAP_WAY")),
		QuitURL:    req.QuitURL,
		TimeExpire: req.TimeExpire,
	}
	return c.client.TradeWapPay(param)
}

func (c *AlipayClient) AppPay(req AlipayTradeRequest) (string, error) {
	if err := req.validate(); err != nil {
		return "", err
	}
	param := alipay.TradeAppPay{
		Trade: c.trade(req, fallback(req.ProductCode, "QUICK_MSECURITY_PAY")),
	}
	return c.client.TradeAppPay(param)
}

func (c *AlipayClient) PreCreate(ctx context.Context, req AlipayTradeRequest) (*alipay.TradePreCreateRsp, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	param := alipay.TradePreCreate{
		Trade: c.trade(req, fallback(req.ProductCode, "FACE_TO_FACE_PAYMENT")),
	}
	return c.client.TradePreCreate(ctx, param)
}

func (c *AlipayClient) Query(ctx context.Context, outTradeNo string) (*alipay.TradeQueryRsp, error) {
	if strings.TrimSpace(outTradeNo) == "" {
		return nil, fmt.Errorf("out trade no is required")
	}
	return c.client.TradeQuery(ctx, alipay.TradeQuery{OutTradeNo: outTradeNo})
}

func (c *AlipayClient) QueryByTradeNo(ctx context.Context, tradeNo string) (*alipay.TradeQueryRsp, error) {
	if strings.TrimSpace(tradeNo) == "" {
		return nil, fmt.Errorf("trade no is required")
	}
	return c.client.TradeQuery(ctx, alipay.TradeQuery{TradeNo: tradeNo})
}

func (c *AlipayClient) Close(ctx context.Context, outTradeNo string) (*alipay.TradeCloseRsp, error) {
	if strings.TrimSpace(outTradeNo) == "" {
		return nil, fmt.Errorf("out trade no is required")
	}
	return c.client.TradeClose(ctx, alipay.TradeClose{OutTradeNo: outTradeNo})
}

func (c *AlipayClient) Refund(ctx context.Context, req AlipayRefundRequest) (*alipay.TradeRefundRsp, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	return c.client.TradeRefund(ctx, alipay.TradeRefund{
		AppAuthToken: req.AppAuthToken,
		OutTradeNo:   req.OutTradeNo,
		TradeNo:      req.TradeNo,
		RefundAmount: YuanFromFen(req.Amount),
		RefundReason: req.Reason,
		OutRequestNo: req.OutRequestNo,
		QueryOptions: req.QueryOptions,
	})
}

func (c *AlipayClient) QueryRefund(ctx context.Context, outTradeNo, outRequestNo string) (*alipay.TradeFastPayRefundQueryRsp, error) {
	if strings.TrimSpace(outTradeNo) == "" {
		return nil, fmt.Errorf("out trade no is required")
	}
	if strings.TrimSpace(outRequestNo) == "" {
		return nil, fmt.Errorf("out request no is required")
	}
	return c.client.TradeFastPayRefundQuery(ctx, alipay.TradeFastPayRefundQuery{
		OutTradeNo:   outTradeNo,
		OutRequestNo: outRequestNo,
	})
}

func (c *AlipayClient) ParseNotify(ctx context.Context, request *http.Request) (*alipay.Notification, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if err := request.ParseForm(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.client.DecodeNotification(ctx, request.Form)
}

func AckAlipayNotification(w http.ResponseWriter) {
	alipay.ACKNotification(w)
}

func (c *AlipayClient) trade(req AlipayTradeRequest, productCode string) alipay.Trade {
	return alipay.Trade{
		NotifyURL:      fallback(req.NotifyURL, c.config.NotifyURL),
		ReturnURL:      fallback(req.ReturnURL, c.config.ReturnURL),
		AppAuthToken:   req.AppAuthToken,
		Subject:        req.Subject,
		OutTradeNo:     req.OutTradeNo,
		TotalAmount:    YuanFromFen(req.Amount),
		ProductCode:    productCode,
		Body:           req.Body,
		TimeoutExpress: req.TimeoutExpress,
		TimeExpire:     req.TimeExpire,
		PassbackParams: req.PassbackParams,
	}
}

func (c AlipayConfig) privateKey() (string, error) {
	if strings.TrimSpace(c.PrivateKey) != "" {
		return c.PrivateKey, nil
	}
	if strings.TrimSpace(c.PrivateKeyPath) == "" {
		return "", fmt.Errorf("alipay private key or private key path is required")
	}
	data, err := os.ReadFile(c.PrivateKeyPath)
	if err != nil {
		return "", fmt.Errorf("read alipay private key: %w", err)
	}
	return string(data), nil
}

func (r AlipayTradeRequest) validate() error {
	switch {
	case strings.TrimSpace(r.Subject) == "":
		return fmt.Errorf("subject is required")
	case strings.TrimSpace(r.OutTradeNo) == "":
		return fmt.Errorf("out trade no is required")
	case r.Amount <= 0:
		return fmt.Errorf("amount must be greater than zero")
	}
	return nil
}

func (r AlipayRefundRequest) validate() error {
	if strings.TrimSpace(r.OutTradeNo) == "" && strings.TrimSpace(r.TradeNo) == "" {
		return fmt.Errorf("out trade no or trade no is required")
	}
	if r.Amount <= 0 {
		return fmt.Errorf("refund amount must be greater than zero")
	}
	return nil
}
