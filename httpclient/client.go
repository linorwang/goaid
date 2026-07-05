package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type client struct {
	httpClient  *http.Client
	middlewares []Middleware
	mu          sync.RWMutex
	config      *clientConfig
}

// New creates a reusable HTTP client.
func New(opts ...ClientOption) Client {
	config := &clientConfig{
		timeout:                30 * time.Second,
		maxIdleConns:           100,
		maxIdleConnsPerHost:    100,
		idleConnTimeout:        90 * time.Second,
		keepAlive:              30 * time.Second,
		tlsHandshakeTimeout:    10 * time.Second,
		forceAttemptHTTP2:      true,
		defaultMaxRetries:      0,
		defaultBackoffStrategy: NewExponentialBackoff(100*time.Millisecond, 5*time.Second),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(config)
		}
	}

	var transport http.RoundTripper
	if config.transport != nil {
		transport = config.transport
	} else {
		base := http.DefaultTransport.(*http.Transport).Clone()
		base.MaxIdleConns = config.maxIdleConns
		base.IdleConnTimeout = config.idleConnTimeout
		base.MaxIdleConnsPerHost = config.maxIdleConnsPerHost
		base.MaxConnsPerHost = config.maxConnsPerHost
		base.ResponseHeaderTimeout = config.responseHeaderTimeout
		base.TLSHandshakeTimeout = config.tlsHandshakeTimeout
		base.ForceAttemptHTTP2 = config.forceAttemptHTTP2

		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: config.keepAlive,
		}
		base.DialContext = dialer.DialContext
		transport = base
	}

	return &client{
		httpClient: &http.Client{
			Timeout:   config.timeout,
			Transport: transport,
		},
		middlewares: make([]Middleware, 0),
		config:      config,
	}
}

func (c *client) Get(ctx context.Context, requestURL string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodGet, requestURL, opts...)
}

func (c *client) Post(ctx context.Context, requestURL string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodPost, requestURL, opts...)
}

func (c *client) Put(ctx context.Context, requestURL string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodPut, requestURL, opts...)
}

func (c *client) Delete(ctx context.Context, requestURL string, opts ...RequestOption) (*http.Response, error) {
	return c.Send(ctx, http.MethodDelete, requestURL, opts...)
}

func (c *client) Send(ctx context.Context, method, requestURL string, opts ...RequestOption) (*http.Response, error) {
	config, err := newRequestConfig(opts...)
	if err != nil {
		return nil, err
	}
	return c.send(ctx, method, requestURL, config)
}

func (c *client) Do(ctx context.Context, method, requestURL string, opts ...RequestOption) (*Response, error) {
	config, err := newRequestConfig(opts...)
	if err != nil {
		return nil, err
	}

	resp, err := c.send(ctx, method, requestURL, config)
	if err != nil {
		return nil, err
	}
	return ReadResponse(resp)
}

func (c *client) Use(middleware ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = append(c.middlewares, middleware...)
}

func (c *client) Clone() Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

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

	middlewares := make([]Middleware, len(c.middlewares))
	copy(middlewares, c.middlewares)

	return &client{
		httpClient:  c.httpClient,
		middlewares: middlewares,
		config:      newConfig,
	}
}

func (c *client) send(ctx context.Context, method, requestURL string, config *RequestConfig) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	finalURL, err := buildURL(requestURL, config.QueryParams)
	if err != nil {
		return nil, err
	}

	var cancel context.CancelFunc
	if config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
	}

	maxRetries := c.config.defaultMaxRetries
	if config.maxRetriesSet {
		maxRetries = config.MaxRetries
	}
	if maxRetries < 0 {
		maxRetries = 0
	}

	handler := c.buildHandler()
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepWithContext(ctx, c.backoff(attempt-1)); err != nil {
				if cancel != nil {
					cancel()
				}
				return nil, err
			}
		}

		req, err := newHTTPRequest(ctx, method, finalURL, config)
		if err != nil {
			if cancel != nil {
				cancel()
			}
			return nil, err
		}

		middlewareCtx := NewContext(req)
		err = handler(middlewareCtx)
		if err != nil {
			lastErr = err
			continue
		}

		resp := middlewareCtx.Response
		if shouldRetryResponse(resp, attempt, maxRetries) {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			drainAndClose(resp.Body)
			continue
		}

		if cancel != nil {
			resp.Body = &cancelOnCloseReadCloser{
				ReadCloser: resp.Body,
				cancel:     cancel,
			}
		}
		return resp, nil
	}

	if cancel != nil {
		cancel()
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("request failed")
	}
	return nil, lastErr
}

func (c *client) buildHandler() Handler {
	middlewares := c.middlewareSnapshot()
	handler := func(ctx *Context) error {
		resp, err := c.httpClient.Do(ctx.Request)
		if err != nil {
			ctx.Error = err
			return err
		}
		ctx.Response = resp
		return nil
	}

	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func (c *client) middlewareSnapshot() []Middleware {
	c.mu.RLock()
	defer c.mu.RUnlock()

	middlewares := make([]Middleware, len(c.middlewares))
	copy(middlewares, c.middlewares)
	return middlewares
}

func (c *client) backoff(retry int) time.Duration {
	if c.config.defaultBackoffStrategy == nil {
		return 0
	}
	return c.config.defaultBackoffStrategy.Next(retry)
}

func newRequestConfig(opts ...RequestOption) (*RequestConfig, error) {
	config := &RequestConfig{
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(config)
		}
	}

	if config.err != nil {
		return nil, config.err
	}
	return config, nil
}

func buildURL(requestURL string, queryParams map[string]string) (string, error) {
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", requestURL, err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL %q: URL must include scheme and host", requestURL)
	}

	query := parsedURL.Query()
	for key, value := range queryParams {
		query.Set(key, value)
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func newHTTPRequest(ctx context.Context, method, requestURL string, config *RequestConfig) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(config.Body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}
	if config.basicAuthSet {
		req.SetBasicAuth(config.basicAuthUsername, config.basicAuthPassword)
	}
	return req, nil
}

func shouldRetryResponse(resp *http.Response, attempt, maxRetries int) bool {
	return resp != nil && resp.StatusCode >= http.StatusInternalServerError && attempt < maxRetries
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	io.Copy(io.Discard, body)
	body.Close()
}

func resetBodyForRetry(req *http.Request) error {
	if req == nil || req.Body == nil {
		return nil
	}
	if req.GetBody == nil {
		return fmt.Errorf("request body cannot be replayed for retry")
	}

	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("reset request body: %w", err)
	}
	req.Body = body
	return nil
}

type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelOnCloseReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
}

// WithTimeout sets a timeout for a single request.
func WithTimeout(timeout time.Duration) RequestOption {
	return func(config *RequestConfig) {
		config.Timeout = timeout
	}
}

// WithHeader sets a request header.
func WithHeader(key, value string) RequestOption {
	return func(config *RequestConfig) {
		config.Headers[key] = value
	}
}

// WithHeaders sets multiple request headers.
func WithHeaders(headers map[string]string) RequestOption {
	return func(config *RequestConfig) {
		for key, value := range headers {
			config.Headers[key] = value
		}
	}
}

// WithQueryParam sets a query parameter.
func WithQueryParam(key, value string) RequestOption {
	return func(config *RequestConfig) {
		config.QueryParams[key] = value
	}
}

// WithQueryParams sets multiple query parameters.
func WithQueryParams(params map[string]string) RequestOption {
	return func(config *RequestConfig) {
		for key, value := range params {
			config.QueryParams[key] = value
		}
	}
}

// WithBody sets the raw request body.
func WithBody(body []byte) RequestOption {
	return func(config *RequestConfig) {
		config.Body = append([]byte(nil), body...)
	}
}

// WithJSON marshals v as JSON and sets Content-Type when it is absent.
func WithJSON(v any) RequestOption {
	return func(config *RequestConfig) {
		body, err := json.Marshal(v)
		if err != nil {
			config.err = fmt.Errorf("marshal JSON body: %w", err)
			return
		}

		config.Body = body
		if !hasHeader(config.Headers, "Content-Type") {
			config.Headers["Content-Type"] = "application/json"
		}
	}
}

// WithBearerToken sets Authorization: Bearer <token> for one request.
func WithBearerToken(token string) RequestOption {
	return func(config *RequestConfig) {
		if token != "" {
			config.Headers["Authorization"] = "Bearer " + token
		}
	}
}

// WithBasicAuth sets HTTP Basic Auth for one request.
func WithBasicAuth(username, password string) RequestOption {
	return func(config *RequestConfig) {
		config.basicAuthUsername = username
		config.basicAuthPassword = password
		config.basicAuthSet = true
	}
}

// WithRetry sets retry count for one request. Use WithRetry(0) to disable client defaults.
func WithRetry(maxRetries int) RequestOption {
	return func(config *RequestConfig) {
		config.MaxRetries = maxRetries
		config.maxRetriesSet = true
	}
}

// ReadAllResponseBody reads and closes resp.Body.
func ReadAllResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}
