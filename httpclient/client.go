package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// client 实现Client接口
type client struct {
	httpClient *http.Client
}

// New 创建一个新的HTTP客户端实例
func New(opts ...ClientOption) Client {
	config := &clientConfig{
		timeout:         30 * time.Second,
		maxIdleConns:    100,
		idleConnTimeout: 90 * time.Second,
	}

	for _, opt := range opts {
		opt(config)
	}

	transport := &http.Transport{
		MaxIdleConns:        config.maxIdleConns,
		IdleConnTimeout:     config.idleConnTimeout,
		MaxIdleConnsPerHost: config.maxIdleConnsPerHost,
	}

	if config.transport != nil {
		transport = config.transport
	}

	httpClient := &http.Client{
		Timeout:   config.timeout,
		Transport: transport,
	}

	return &client{
		httpClient: httpClient,
	}
}

// Get 发送GET请求
func (c *client) Get(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodGet, url, opts...)
}

// Post 发送POST请求
func (c *client) Post(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodPost, url, opts...)
}

// Put 发送PUT请求
func (c *client) Put(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodPut, url, opts...)
}

// Delete 发送DELETE请求
func (c *client) Delete(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodDelete, url, opts...)
}

// Send 发送自定义方法请求
func (c *client) Send(ctx context.Context, method, requestURL string, opts ...RequestOption) (*http.Response, error) {
	config := &RequestConfig{
		Timeout:     0, // 使用客户端默认超时
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
	}

	// 应用请求选项
	for _, opt := range opts {
		opt(config)
	}

	// 构建带查询参数的URL
	parsedURL, err := url.ParseRequestURI(requestURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	query := parsedURL.Query()
	// 添加查询参数
	for key, value := range config.QueryParams {
		query.Set(key, value)
	}
	parsedURL.RawQuery = query.Encode()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), bytes.NewReader(config.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// 如果有设置特定超时时间，则使用它
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// WithTimeout 设置请求超时时间
func WithTimeout(timeout time.Duration) RequestOption {
	return func(config *RequestConfig) {
		config.Timeout = timeout
	}
}

// WithHeader 设置请求头
func WithHeader(key, value string) RequestOption {
	return func(config *RequestConfig) {
		config.Headers[key] = value
	}
}

// WithHeaders 批量设置请求头
func WithHeaders(headers map[string]string) RequestOption {
	return func(config *RequestConfig) {
		for key, value := range headers {
			config.Headers[key] = value
		}
	}
}

// WithQueryParam 设置查询参数
func WithQueryParam(key, value string) RequestOption {
	return func(config *RequestConfig) {
		config.QueryParams[key] = value
	}
}

// WithQueryParams 批量设置查询参数
func WithQueryParams(params map[string]string) RequestOption {
	return func(config *RequestConfig) {
		for key, value := range params {
			config.QueryParams[key] = value
		}
	}
}

// WithBody 设置请求体
func WithBody(body []byte) RequestOption {
	return func(config *RequestConfig) {
		config.Body = body
	}
}

// ReadAllResponseBody 读取并关闭响应体
func ReadAllResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
