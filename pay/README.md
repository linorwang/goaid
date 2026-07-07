# pay 支付接入

`pay` 封装了微信支付、支付宝支付和云闪付/银联 ACP 支付的常用入口。金额统一用“分”的 `int64`，支付宝会自动转成元字符串，微信和云闪付直接使用分。

## 依赖版本

- 微信支付：`github.com/wechatpay-apiv3/wechatpay-go v0.2.21`
- 支付宝：`github.com/smartwalle/alipay/v3 v3.2.30`
- 云闪付/银联证书读取：`software.sslmate.com/src/go-pkcs12 v0.7.3`

## 微信支付

```go
wx, err := pay.NewWechatClient(ctx, pay.WechatConfig{
	AppID:                      "wx...",
	MchID:                      "1900000000",
	MchCertificateSerialNumber: "证书序列号",
	MchAPIv3Key:                "APIv3Key",
	PrivateKeyPath:             "/secure/apiclient_key.pem",
	NotifyURL:                  "https://example.com/pay/wechat/notify",
	RefundNotifyURL:            "https://example.com/pay/wechat/refund_notify",
})
```

JSAPI 下单：

```go
resp, _, err := wx.JSAPIPrepay(ctx, pay.WechatPrepayRequest{
	Description: "测试商品",
	OutTradeNo:  "ORDER202607070001",
	Amount:      100,
	PayerOpenID: "用户 openid",
})
```

Native / App / H5：

```go
nativeResp, _, err := wx.NativePrepay(ctx, pay.WechatPrepayRequest{
	Description: "测试商品",
	OutTradeNo:  "ORDER202607070002",
	Amount:      100,
})

appResp, _, err := wx.AppPrepay(ctx, pay.WechatPrepayRequest{
	Description: "测试商品",
	OutTradeNo:  "ORDER202607070003",
	Amount:      100,
})

h5Resp, _, err := wx.H5Prepay(ctx, pay.WechatPrepayRequest{
	Description:   "测试商品",
	OutTradeNo:    "ORDER202607070004",
	Amount:        100,
	PayerClientIP: "127.0.0.1",
	H5Type:        "Wap",
})
```

查单、关单、退款：

```go
order, _, err := wx.QueryOrderByOutTradeNo(ctx, "ORDER202607070001")
_, err = wx.CloseOrder(ctx, "ORDER202607070001")

refund, _, err := wx.Refund(ctx, pay.WechatRefundRequest{
	OutTradeNo:  "ORDER202607070001",
	OutRefundNo: "REFUND202607070001",
	Total:       100,
	Refund:      100,
})
```

通知：

```go
func WechatNotify(w http.ResponseWriter, r *http.Request) {
	transaction, _, err := wx.ParseTransactionNotify(r.Context(), r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = transaction
	pay.AckWechatNotification(w)
}
```

## 支付宝

```go
ali, err := pay.NewAlipayClient(pay.AlipayConfig{
	AppID:                  "2021000000000000",
	PrivateKeyPath:         "/secure/alipay_app_private_key.txt",
	Production:             true,
	NotifyURL:              "https://example.com/pay/alipay/notify",
	ReturnURL:              "https://example.com/pay/alipay/return",
	AppCertPublicKeyPath:    "/secure/appCertPublicKey.crt",
	AlipayRootCertPath:      "/secure/alipayRootCert.crt",
	AlipayCertPublicKeyPath: "/secure/alipayCertPublicKey_RSA2.crt",
})
```

PC / WAP / App / 当面付二维码：

```go
pageURL, err := ali.PagePay(pay.AlipayTradeRequest{
	Subject:    "测试商品",
	OutTradeNo: "ORDER202607070005",
	Amount:     100,
})

wapURL, err := ali.WapPay(pay.AlipayTradeRequest{
	Subject:    "测试商品",
	OutTradeNo: "ORDER202607070006",
	Amount:     100,
	QuitURL:    "https://example.com/cancel",
})

orderString, err := ali.AppPay(pay.AlipayTradeRequest{
	Subject:    "测试商品",
	OutTradeNo: "ORDER202607070007",
	Amount:     100,
})

qr, err := ali.PreCreate(ctx, pay.AlipayTradeRequest{
	Subject:    "测试商品",
	OutTradeNo: "ORDER202607070008",
	Amount:     100,
})
```

查单、退款、通知：

```go
order, err := ali.Query(ctx, "ORDER202607070005")

refund, err := ali.Refund(ctx, pay.AlipayRefundRequest{
	OutTradeNo:   "ORDER202607070005",
	OutRequestNo: "REFUND202607070005",
	Amount:       100,
	Reason:       "用户退款",
})

func AlipayNotify(w http.ResponseWriter, r *http.Request) {
	notification, err := ali.ParseNotify(r.Context(), r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = notification
	pay.AckAlipayNotification(w)
}
```

## 云闪付 / 银联 ACP

云闪付服务端接入使用银联 ACP 网关协议。该包内置 RSA-SHA256 签名、验签、前台支付表单、App 支付 `tn` 获取、查单和退款。生产环境建议使用 `.pfx` 商户签名证书和银联验签证书目录。

```go
union, err := pay.NewUnionPayClient(pay.UnionPayConfig{
	MerchantID:            "777290058110048",
	PrivateKeyPFXPath:     "/secure/acp_prod_sign.pfx",
	PrivateKeyPFXPassword: "证书密码",
	VerifyCertDir:         "/secure/unionpay_verify_certs",
	FrontURL:              "https://example.com/pay/unionpay/return",
	BackURL:               "https://example.com/pay/unionpay/notify",
})
```

前台网页支付：

```go
form, err := union.BuildFrontPayForm(pay.UnionPayTradeRequest{
	OrderID: "ORDER202607070009",
	Amount:  100,
})
html := form.HTML()
```

App 支付获取 `tn`：

```go
resp, err := union.AppPay(ctx, pay.UnionPayTradeRequest{
	OrderID: "ORDER202607070010",
	Amount:  100,
})
tn := resp.TN()
```

查单、退款：

```go
query, err := union.Query(ctx, pay.UnionPayQueryRequest{
	OrderID: "ORDER202607070010",
	TxnTime: "20260707153000",
})

refund, err := union.Refund(ctx, pay.UnionPayRefundRequest{
	OrderID:         "REFUND202607070010",
	OriginalQueryID: query.QueryID(),
	Amount:          100,
})
```

通知验签：

```go
func UnionPayNotify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	params, err := union.VerifyNotification(r.Form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = params
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
```

## 说明

- 私钥、证书、APIv3Key 不要提交到仓库。
- 微信回调读取后 SDK 会恢复 `Request.Body`，业务层仍然可以再次读取。
- 支付宝 `NotifyURL` 和 `ReturnURL` 不建议带查询参数，避免验签时混入自定义参数。
- 云闪付验签需要配置银联公钥证书，生产环境不要跳过响应/通知验签。

