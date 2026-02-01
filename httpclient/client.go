package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// client 实现Client接口
type client struct {
	httpClient  *http.Client
	middlewares []Middleware
	mu          sync.RWMutex
	config      *clientConfig
}

// New 创建一个新的HTTP客户端实例
func New(opts ...ClientOption) Client {
	config := &clientConfig{
		timeout:                30 * time.Second,
		maxIdleConns:           100,
		maxIdleConnsPerHost:    100,
		idleConnTimeout:        90 * time.Second,
		keepAlive:              30 * time.Second,
		defaultMaxRetries:      0,
		defaultBackoffStrategy: NewExponentialBackoff(100*time.Millisecond, 5*time.Second),
	}

	for _, opt := range opts {
		opt(config)
	}

	transport := &http.Transport{
		MaxIdleConns:          config.maxIdleConns,
		IdleConnTimeout:       config.idleConnTimeout,
		MaxIdleConnsPerHost:   config.maxIdleConnsPerHost,
		MaxConnsPerHost:       config.maxConnsPerHost,
		ResponseHeaderTimeout: config.responseHeaderTimeout,
		TLSHandshakeTimeout:   config.tlsHandshakeTimeout,
		ForceAttemptHTTP2:     config.forceAttemptHTTP2,
	}

	if config.keepAlive > 0 {
		transport.IdleConnTimeout = config.keepAlive
	}

	if config.transport != nil {
		transport = config.transport
	}

	httpClient := &http.Client{
		Timeout:   config.timeout,
		Transport: transport,
	}

	return &client{
		httpClient:  httpClient,
		middlewares: make([]Middleware, 0),
		config:      config,
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
	resp, err := c.Do(ctx, method, requestURL, opts...)
	if err != nil {
		return nil, err
	}
	return resp.Response, nil
}

// Do 发送请求并返回包装后的响应
func (c *client) Do(ctx context.Context, method, requestURL string, opts ...RequestOption) (*Response, error) {
	config := &RequestConfig{
		Timeout:     0,
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

	// 处理重试
	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = c.config.defaultMaxRetries
	}

	var lastResp *http.Response
	var lastErr error

	for retry := 0; retry <= maxRetries; retry++ {
		if retry > 0 {
			// 执行退避
			backoff := c.config.defaultBackoffStrategy.Next(retry - 1)
			time.Sleep(backoff)
		}

		// 使用中间件处理请求
		middlewareCtx := NewContext(req)
		handler := c.buildHandler()

		err = handler(middlewareCtx)
		if err != nil {
			lastErr = err
			continue
		}

		lastResp = middlewareCtx.Response
		if lastResp != nil && lastResp.StatusCode >= 500 && retry < maxRetries {
			// 服务器错误，继续重试
			lastErr = fmt.Errorf("server error: %d", lastResp.StatusCode)
			continue
		}

		// 成功或客户端错误，不再重试
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return ReadResponse(lastResp)
}

// Use 添加中间件
func (c *client) Use(middleware ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = append(c.middlewares, middleware...)
}

// Clone 克隆客户端
func (c *client) Clone() Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 克隆配置
	newConfig := &clientConfig{
		timeout:                c.config.timeout,
		maxIdleConns:           c.config.maxIdleConns,
		maxIdleConnsPerHost:    c.config.maxIdleConnsPerHost,
		idleConnTimeout:        c.config.idleConnTimeout,
		transport:              c.config.transport,
		maxConnsPerHost:        c.config.maxConnsPerHost,
		responseHeaderTimeout:  c.config.responseHeaderTimeout,
		tlsHandshakeTimeout:    c.config.tlsHandshakeTimeout,
		keepAlive:              c.config.keepAlive,
		forceAttemptHTTP2:      c.config.forceAttemptHTTP2,
		defaultBackoffStrategy: c.config.defaultBackoffStrategy,
		defaultMaxRetries:      c.config.defaultMaxRetries,
	}

	// 克隆中间件
	middlewares := make([]Middleware, len(c.middlewares))
	copy(middlewares, c.middlewares)

	return &client{
		httpClient:  c.httpClient,
		middlewares: middlewares,
		config:      newConfig,
	}
}

// buildHandler 构建中间件链
func (c *client) buildHandler() Handler {
	// 从后往前构建中间件链
	handler := func(ctx *Context) error {
		resp, err := c.httpClient.Do(ctx.Request)
		if err != nil {
			ctx.Error = err
			return err
		}
		ctx.Response = resp
		return nil
	}

	// 应用中间件
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}

	return handler
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

// WithRetry 设置重试次数
func WithRetry(maxRetries int) RequestOption {
	return func(config *RequestConfig) {
		config.MaxRetries = maxRetries
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
