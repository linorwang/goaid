package pay

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	wxapp "github.com/wechatpay-apiv3/wechatpay-go/services/payments/app"
	wxh5 "github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	wxjsapi "github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	wxnative "github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

type WechatConfig struct {
	AppID                      string
	MchID                      string
	MchCertificateSerialNumber string
	MchAPIv3Key                string
	PrivateKeyPEM              string
	PrivateKeyPath             string
	NotifyURL                  string
	RefundNotifyURL            string
}

type WechatClient struct {
	config        WechatConfig
	client        *core.Client
	notifyHandler *notify.Handler
	jsapi         wxjsapi.JsapiApiService
	native        wxnative.NativeApiService
	app           wxapp.AppApiService
	h5            wxh5.H5ApiService
	refunds       refunddomestic.RefundsApiService
}

type WechatPrepayRequest struct {
	Description   string
	OutTradeNo    string
	Amount        int64
	Currency      string
	NotifyURL     string
	TimeExpire    *time.Time
	Attach        string
	GoodsTag      string
	LimitPay      []string
	SupportFapiao *bool
	PayerOpenID   string
	PayerClientIP string
	H5Type        string
	H5AppName     string
	H5AppURL      string
	H5BundleID    string
	H5PackageName string
}

type WechatRefundRequest struct {
	TransactionID string
	OutTradeNo    string
	OutRefundNo   string
	Reason        string
	NotifyURL     string
	Total         int64
	Refund        int64
	Currency      string
}

type WechatRefundNotification struct {
	MchID               string                          `json:"mchid"`
	OutTradeNo          string                          `json:"out_trade_no"`
	TransactionID       string                          `json:"transaction_id"`
	OutRefundNo         string                          `json:"out_refund_no"`
	RefundID            string                          `json:"refund_id"`
	RefundStatus        string                          `json:"refund_status"`
	SuccessTime         string                          `json:"success_time"`
	UserReceivedAccount string                          `json:"user_received_account"`
	Amount              *WechatRefundNotificationAmount `json:"amount"`
}

type WechatRefundNotificationAmount struct {
	Total       int64 `json:"total"`
	Refund      int64 `json:"refund"`
	PayerTotal  int64 `json:"payer_total"`
	PayerRefund int64 `json:"payer_refund"`
}

func NewWechatClient(ctx context.Context, cfg WechatConfig) (*WechatClient, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	privateKey, err := cfg.loadPrivateKey()
	if err != nil {
		return nil, err
	}

	coreClient, err := core.NewClient(ctx, option.WithWechatPayAutoAuthCipher(
		cfg.MchID,
		cfg.MchCertificateSerialNumber,
		privateKey,
		cfg.MchAPIv3Key,
	))
	if err != nil {
		return nil, fmt.Errorf("new wechat client: %w", err)
	}

	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(cfg.MchID)
	notifyHandler, err := notify.NewRSANotifyHandler(
		cfg.MchAPIv3Key,
		verifiers.NewSHA256WithRSAVerifier(certificateVisitor),
	)
	if err != nil {
		return nil, fmt.Errorf("new wechat notify handler: %w", err)
	}

	client := &WechatClient{
		config:        cfg,
		client:        coreClient,
		notifyHandler: notifyHandler,
	}
	client.jsapi = wxjsapi.JsapiApiService{Client: coreClient}
	client.native = wxnative.NativeApiService{Client: coreClient}
	client.app = wxapp.AppApiService{Client: coreClient}
	client.h5 = wxh5.H5ApiService{Client: coreClient}
	client.refunds = refunddomestic.RefundsApiService{Client: coreClient}
	return client, nil
}

func (c *WechatClient) SDKClient() *core.Client {
	return c.client
}

func (c *WechatClient) Config() WechatConfig {
	return c.config
}

func (c *WechatClient) JSAPIPrepay(ctx context.Context, req WechatPrepayRequest) (*wxjsapi.PrepayWithRequestPaymentResponse, *core.APIResult, error) {
	if strings.TrimSpace(req.PayerOpenID) == "" {
		return nil, nil, fmt.Errorf("payer openid is required for wechat jsapi pay")
	}
	sdkReq, err := c.jsapiPrepayRequest(req)
	if err != nil {
		return nil, nil, err
	}
	return c.jsapi.PrepayWithRequestPayment(ctx, sdkReq)
}

func (c *WechatClient) NativePrepay(ctx context.Context, req WechatPrepayRequest) (*wxnative.PrepayResponse, *core.APIResult, error) {
	sdkReq, err := c.nativePrepayRequest(req)
	if err != nil {
		return nil, nil, err
	}
	return c.native.Prepay(ctx, sdkReq)
}

func (c *WechatClient) AppPrepay(ctx context.Context, req WechatPrepayRequest) (*wxapp.PrepayWithRequestPaymentResponse, *core.APIResult, error) {
	sdkReq, err := c.appPrepayRequest(req)
	if err != nil {
		return nil, nil, err
	}
	return c.app.PrepayWithRequestPayment(ctx, sdkReq)
}

func (c *WechatClient) H5Prepay(ctx context.Context, req WechatPrepayRequest) (*wxh5.PrepayResponse, *core.APIResult, error) {
	if strings.TrimSpace(req.PayerClientIP) == "" {
		return nil, nil, fmt.Errorf("payer client ip is required for wechat h5 pay")
	}
	sdkReq, err := c.h5PrepayRequest(req)
	if err != nil {
		return nil, nil, err
	}
	return c.h5.Prepay(ctx, sdkReq)
}

func (c *WechatClient) QueryOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*payments.Transaction, *core.APIResult, error) {
	if strings.TrimSpace(outTradeNo) == "" {
		return nil, nil, fmt.Errorf("out trade no is required")
	}
	return c.jsapi.QueryOrderByOutTradeNo(ctx, wxjsapi.QueryOrderByOutTradeNoRequest{
		OutTradeNo: stringPtr(outTradeNo),
		Mchid:      stringPtr(c.config.MchID),
	})
}

func (c *WechatClient) QueryOrderByTransactionID(ctx context.Context, transactionID string) (*payments.Transaction, *core.APIResult, error) {
	if strings.TrimSpace(transactionID) == "" {
		return nil, nil, fmt.Errorf("transaction id is required")
	}
	return c.jsapi.QueryOrderById(ctx, wxjsapi.QueryOrderByIdRequest{
		TransactionId: stringPtr(transactionID),
		Mchid:         stringPtr(c.config.MchID),
	})
}

func (c *WechatClient) CloseOrder(ctx context.Context, outTradeNo string) (*core.APIResult, error) {
	if strings.TrimSpace(outTradeNo) == "" {
		return nil, fmt.Errorf("out trade no is required")
	}
	return c.jsapi.CloseOrder(ctx, wxjsapi.CloseOrderRequest{
		OutTradeNo: stringPtr(outTradeNo),
		Mchid:      stringPtr(c.config.MchID),
	})
}

func (c *WechatClient) Refund(ctx context.Context, req WechatRefundRequest) (*refunddomestic.Refund, *core.APIResult, error) {
	if strings.TrimSpace(req.OutRefundNo) == "" {
		return nil, nil, fmt.Errorf("out refund no is required")
	}
	if strings.TrimSpace(req.TransactionID) == "" && strings.TrimSpace(req.OutTradeNo) == "" {
		return nil, nil, fmt.Errorf("transaction id or out trade no is required")
	}
	if req.Total <= 0 || req.Refund <= 0 {
		return nil, nil, fmt.Errorf("total and refund amount must be greater than zero")
	}

	notifyURL := fallback(req.NotifyURL, c.config.RefundNotifyURL)
	sdkReq := refunddomestic.CreateRequest{
		TransactionId: optionalString(req.TransactionID),
		OutTradeNo:    optionalString(req.OutTradeNo),
		OutRefundNo:   stringPtr(req.OutRefundNo),
		Reason:        optionalString(req.Reason),
		NotifyUrl:     optionalString(notifyURL),
		Amount: &refunddomestic.AmountReq{
			Refund:   int64Ptr(req.Refund),
			Total:    int64Ptr(req.Total),
			Currency: stringPtr(fallback(req.Currency, defaultCurrencyCNY)),
		},
	}
	return c.refunds.Create(ctx, sdkReq)
}

func (c *WechatClient) QueryRefundByOutRefundNo(ctx context.Context, outRefundNo string) (*refunddomestic.Refund, *core.APIResult, error) {
	if strings.TrimSpace(outRefundNo) == "" {
		return nil, nil, fmt.Errorf("out refund no is required")
	}
	return c.refunds.QueryByOutRefundNo(ctx, refunddomestic.QueryByOutRefundNoRequest{
		OutRefundNo: stringPtr(outRefundNo),
	})
}

func (c *WechatClient) ParseTransactionNotify(ctx context.Context, request *http.Request) (*payments.Transaction, *notify.Request, error) {
	transaction := new(payments.Transaction)
	notifyRequest, err := c.ParseNotify(ctx, request, transaction)
	if err != nil {
		return nil, notifyRequest, err
	}
	return transaction, notifyRequest, nil
}

func (c *WechatClient) ParseRefundNotify(ctx context.Context, request *http.Request) (*WechatRefundNotification, *notify.Request, error) {
	refund := new(WechatRefundNotification)
	notifyRequest, err := c.ParseNotify(ctx, request, refund)
	if err != nil {
		return nil, notifyRequest, err
	}
	return refund, notifyRequest, nil
}

func (c *WechatClient) ParseNotify(ctx context.Context, request *http.Request, dest any) (*notify.Request, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if dest == nil {
		return nil, fmt.Errorf("dest cannot be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.notifyHandler.ParseNotifyRequest(ctx, request, dest)
}

func AckWechatNotification(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"code":"SUCCESS","message":"OK"}`))
}

func (c WechatConfig) validate() error {
	switch {
	case strings.TrimSpace(c.AppID) == "":
		return fmt.Errorf("wechat app id is required")
	case strings.TrimSpace(c.MchID) == "":
		return fmt.Errorf("wechat mch id is required")
	case strings.TrimSpace(c.MchCertificateSerialNumber) == "":
		return fmt.Errorf("wechat merchant certificate serial number is required")
	case strings.TrimSpace(c.MchAPIv3Key) == "":
		return fmt.Errorf("wechat api v3 key is required")
	case strings.TrimSpace(c.PrivateKeyPEM) == "" && strings.TrimSpace(c.PrivateKeyPath) == "":
		return fmt.Errorf("wechat private key pem or path is required")
	}
	return nil
}

func (c WechatConfig) loadPrivateKey() (*rsa.PrivateKey, error) {
	if strings.TrimSpace(c.PrivateKeyPEM) != "" {
		return utils.LoadPrivateKey(c.PrivateKeyPEM)
	}
	return utils.LoadPrivateKeyWithPath(c.PrivateKeyPath)
}

func (c *WechatClient) validatePrepay(req WechatPrepayRequest) (string, error) {
	switch {
	case strings.TrimSpace(req.Description) == "":
		return "", fmt.Errorf("description is required")
	case strings.TrimSpace(req.OutTradeNo) == "":
		return "", fmt.Errorf("out trade no is required")
	case req.Amount <= 0:
		return "", fmt.Errorf("amount must be greater than zero")
	}
	notifyURL := fallback(req.NotifyURL, c.config.NotifyURL)
	if strings.TrimSpace(notifyURL) == "" {
		return "", fmt.Errorf("notify url is required")
	}
	return notifyURL, nil
}

func (c *WechatClient) jsapiPrepayRequest(req WechatPrepayRequest) (wxjsapi.PrepayRequest, error) {
	notifyURL, err := c.validatePrepay(req)
	if err != nil {
		return wxjsapi.PrepayRequest{}, err
	}
	return wxjsapi.PrepayRequest{
		Appid:         stringPtr(c.config.AppID),
		Mchid:         stringPtr(c.config.MchID),
		Description:   stringPtr(req.Description),
		OutTradeNo:    stringPtr(req.OutTradeNo),
		TimeExpire:    req.TimeExpire,
		Attach:        optionalString(req.Attach),
		NotifyUrl:     stringPtr(notifyURL),
		GoodsTag:      optionalString(req.GoodsTag),
		LimitPay:      req.LimitPay,
		SupportFapiao: req.SupportFapiao,
		Amount: &wxjsapi.Amount{
			Total:    int64Ptr(req.Amount),
			Currency: optionalString(fallback(req.Currency, defaultCurrencyCNY)),
		},
		Payer: &wxjsapi.Payer{Openid: stringPtr(req.PayerOpenID)},
	}, nil
}

func (c *WechatClient) nativePrepayRequest(req WechatPrepayRequest) (wxnative.PrepayRequest, error) {
	notifyURL, err := c.validatePrepay(req)
	if err != nil {
		return wxnative.PrepayRequest{}, err
	}
	return wxnative.PrepayRequest{
		Appid:         stringPtr(c.config.AppID),
		Mchid:         stringPtr(c.config.MchID),
		Description:   stringPtr(req.Description),
		OutTradeNo:    stringPtr(req.OutTradeNo),
		TimeExpire:    req.TimeExpire,
		Attach:        optionalString(req.Attach),
		NotifyUrl:     stringPtr(notifyURL),
		GoodsTag:      optionalString(req.GoodsTag),
		LimitPay:      req.LimitPay,
		SupportFapiao: req.SupportFapiao,
		Amount: &wxnative.Amount{
			Total:    int64Ptr(req.Amount),
			Currency: optionalString(fallback(req.Currency, defaultCurrencyCNY)),
		},
	}, nil
}

func (c *WechatClient) appPrepayRequest(req WechatPrepayRequest) (wxapp.PrepayRequest, error) {
	notifyURL, err := c.validatePrepay(req)
	if err != nil {
		return wxapp.PrepayRequest{}, err
	}
	return wxapp.PrepayRequest{
		Appid:         stringPtr(c.config.AppID),
		Mchid:         stringPtr(c.config.MchID),
		Description:   stringPtr(req.Description),
		OutTradeNo:    stringPtr(req.OutTradeNo),
		TimeExpire:    req.TimeExpire,
		Attach:        optionalString(req.Attach),
		NotifyUrl:     stringPtr(notifyURL),
		GoodsTag:      optionalString(req.GoodsTag),
		LimitPay:      req.LimitPay,
		SupportFapiao: req.SupportFapiao,
		Amount: &wxapp.Amount{
			Total:    int64Ptr(req.Amount),
			Currency: optionalString(fallback(req.Currency, defaultCurrencyCNY)),
		},
	}, nil
}

func (c *WechatClient) h5PrepayRequest(req WechatPrepayRequest) (wxh5.PrepayRequest, error) {
	notifyURL, err := c.validatePrepay(req)
	if err != nil {
		return wxh5.PrepayRequest{}, err
	}
	h5Type := fallback(req.H5Type, "Wap")
	return wxh5.PrepayRequest{
		Appid:         stringPtr(c.config.AppID),
		Mchid:         stringPtr(c.config.MchID),
		Description:   stringPtr(req.Description),
		OutTradeNo:    stringPtr(req.OutTradeNo),
		TimeExpire:    req.TimeExpire,
		Attach:        optionalString(req.Attach),
		NotifyUrl:     stringPtr(notifyURL),
		GoodsTag:      optionalString(req.GoodsTag),
		LimitPay:      req.LimitPay,
		SupportFapiao: req.SupportFapiao,
		Amount: &wxh5.Amount{
			Total:    int64Ptr(req.Amount),
			Currency: optionalString(fallback(req.Currency, defaultCurrencyCNY)),
		},
		SceneInfo: &wxh5.SceneInfo{
			PayerClientIp: stringPtr(req.PayerClientIP),
			H5Info: &wxh5.H5Info{
				Type:        stringPtr(h5Type),
				AppName:     optionalString(req.H5AppName),
				AppUrl:      optionalString(req.H5AppURL),
				BundleId:    optionalString(req.H5BundleID),
				PackageName: optionalString(req.H5PackageName),
			},
		},
	}, nil
}
