package httpclient

import (
	"net/http"
	"time"
)

// ClientOption 客户端配置选项
type ClientOption func(*clientConfig)

// clientConfig 客户端配置
type clientConfig struct {
	timeout                time.Duration
	maxIdleConns           int
	maxIdleConnsPerHost    int
	idleConnTimeout        time.Duration
	transport              *http.Transport
	maxConnsPerHost        int             // 新增：每个主机最大连接数
	responseHeaderTimeout  time.Duration   // 新增：响应头超时
	tlsHandshakeTimeout    time.Duration   // 新增：TLS握手超时
	keepAlive              time.Duration   // 新增：Keep-Alive超时
	forceAttemptHTTP2      bool            // 新增：强制尝试HTTP/2
	defaultBackoffStrategy BackoffStrategy // 新增：默认退避策略
	defaultMaxRetries      int             // 新增：默认最大重试次数
}

// WithClientTimeout 设置客户端超时时间
func WithClientTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

// WithMaxIdleConns 设置最大空闲连接数
func WithMaxIdleConns(maxIdleConns int) ClientOption {
	return func(c *clientConfig) {
		c.maxIdleConns = maxIdleConns
	}
}

// WithMaxIdleConnsPerHost 设置每个主机的最大空闲连接数
func WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) ClientOption {
	return func(c *clientConfig) {
		c.maxIdleConnsPerHost = maxIdleConnsPerHost
	}
}

// WithIdleConnTimeout 设置空闲连接超时时间
func WithIdleConnTimeout(idleConnTimeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.idleConnTimeout = idleConnTimeout
	}
}

// WithTransport 设置自定义传输层
func WithTransport(transport *http.Transport) ClientOption {
	return func(c *clientConfig) {
		c.transport = transport
	}
}

// WithMaxConnsPerHost 设置每个主机的最大连接数
func WithMaxConnsPerHost(maxConnsPerHost int) ClientOption {
	return func(c *clientConfig) {
		c.maxConnsPerHost = maxConnsPerHost
	}
}

// WithResponseHeaderTimeout 设置响应头超时时间
func WithResponseHeaderTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.responseHeaderTimeout = timeout
	}
}

// WithTLSHandshakeTimeout 设置TLS握手超时时间
func WithTLSHandshakeTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.tlsHandshakeTimeout = timeout
	}
}

// WithKeepAlive 设置Keep-Alive超时时间
func WithKeepAlive(keepAlive time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.keepAlive = keepAlive
	}
}

// WithForceAttemptHTTP22 设置是否强制尝试HTTP/2
func WithForceAttemptHTTP2(force bool) ClientOption {
	return func(c *clientConfig) {
		c.forceAttemptHTTP2 = force
	}
}

// WithDefaultBackoffStrategy 设置默认退避策略
func WithDefaultBackoffStrategy(strategy BackoffStrategy) ClientOption {
	return func(c *clientConfig) {
		c.defaultBackoffStrategy = strategy
	}
}

// WithDefaultMaxRetries 设置默认最大重试次数
func WithDefaultMaxRetries(maxRetries int) ClientOption {
	return func(c *clientConfig) {
		c.defaultMaxRetries = maxRetries
	}
}
