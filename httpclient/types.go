package httpclient

import (
	"context"
	"net/http"
	"time"
)

// RequestOption 定义请求选项
type RequestOption func(*RequestConfig)

// RequestConfig 请求配置
type RequestConfig struct {
	Timeout   time.Duration
	Headers   map[string]string
	QueryParams map[string]string
	Body      []byte
}

// Client HTTP客户端接口
type Client interface {
	// Get 发送GET请求
	Get(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	
	// Post 发送POST请求
	Post(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	
	// Put 发送PUT请求
	Put(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	
	// Delete 发送DELETE请求
	Delete(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	
	// Send 发送自定义方法请求
	Send(ctx context.Context, method, url string, opts ...RequestOption) (*http.Response, error)
}

// ResponseWrapper 响应包装器
type ResponseWrapper struct {
	*http.Response
	BodyBytes []byte
}