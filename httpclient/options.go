package httpclient

import (
	"net/http"
	"time"
)

// ClientOption 客户端配置选项
type ClientOption func(*clientConfig)

// clientConfig 客户端配置
type clientConfig struct {
	timeout             time.Duration
	maxIdleConns        int
	maxIdleConnsPerHost int
	idleConnTimeout     time.Duration
	transport           *http.Transport
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