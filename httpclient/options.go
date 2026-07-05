package httpclient

import (
	"net/http"
	"time"
)

// ClientOption configures a reusable client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	timeout                time.Duration
	maxIdleConns           int
	maxIdleConnsPerHost    int
	idleConnTimeout        time.Duration
	transport              *http.Transport
	maxConnsPerHost        int
	responseHeaderTimeout  time.Duration
	tlsHandshakeTimeout    time.Duration
	keepAlive              time.Duration
	forceAttemptHTTP2      bool
	defaultBackoffStrategy BackoffStrategy
	defaultMaxRetries      int
}

// WithClientTimeout sets http.Client.Timeout.
func WithClientTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(maxIdleConns int) ClientOption {
	return func(c *clientConfig) {
		c.maxIdleConns = maxIdleConns
	}
}

// WithMaxIdleConnsPerHost sets the maximum idle connections per host.
func WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) ClientOption {
	return func(c *clientConfig) {
		c.maxIdleConnsPerHost = maxIdleConnsPerHost
	}
}

// WithIdleConnTimeout sets how long idle connections stay open.
func WithIdleConnTimeout(idleConnTimeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.idleConnTimeout = idleConnTimeout
	}
}

// WithTransport uses a custom transport.
func WithTransport(transport *http.Transport) ClientOption {
	return func(c *clientConfig) {
		c.transport = transport
	}
}

// WithMaxConnsPerHost sets the maximum total connections per host.
func WithMaxConnsPerHost(maxConnsPerHost int) ClientOption {
	return func(c *clientConfig) {
		c.maxConnsPerHost = maxConnsPerHost
	}
}

// WithResponseHeaderTimeout sets the timeout for waiting for response headers.
func WithResponseHeaderTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.responseHeaderTimeout = timeout
	}
}

// WithTLSHandshakeTimeout sets the TLS handshake timeout.
func WithTLSHandshakeTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.tlsHandshakeTimeout = timeout
	}
}

// WithKeepAlive sets TCP keep-alive duration on the default transport.
func WithKeepAlive(keepAlive time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.keepAlive = keepAlive
	}
}

// WithForceAttemptHTTP2 controls whether the default transport attempts HTTP/2.
func WithForceAttemptHTTP2(force bool) ClientOption {
	return func(c *clientConfig) {
		c.forceAttemptHTTP2 = force
	}
}

// WithDefaultBackoffStrategy sets the client-level retry backoff strategy.
func WithDefaultBackoffStrategy(strategy BackoffStrategy) ClientOption {
	return func(c *clientConfig) {
		c.defaultBackoffStrategy = strategy
	}
}

// WithDefaultMaxRetries sets the client-level retry count.
func WithDefaultMaxRetries(maxRetries int) ClientOption {
	return func(c *clientConfig) {
		c.defaultMaxRetries = maxRetries
	}
}
