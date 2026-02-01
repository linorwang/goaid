package httpclient

import (
	"context"
	"fmt"
	"log"
	"time"
)

// LoggerMiddleware 日志中间件
type LoggerMiddleware struct {
	logger Logger
}

// Logger 日志接口
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

// DefaultLogger 默认日志实现
type DefaultLogger struct{}

func (l *DefaultLogger) Debugf(format string, args ...any) {
	log.Printf("[DEBUG] "+format, args...)
}

func (l *DefaultLogger) Infof(format string, args ...any) {
	log.Printf("[INFO] "+format, args...)
}

func (l *DefaultLogger) Errorf(format string, args ...any) {
	log.Printf("[ERROR] "+format, args...)
}

// NewLoggerMiddleware 创建日志中间件
func NewLoggerMiddleware(logger Logger) Middleware {
	lm := &LoggerMiddleware{
		logger: logger,
	}
	if logger == nil {
		lm.logger = &DefaultLogger{}
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()

			lm.logger.Debugf("Request: %s %s", ctx.Request.Method, ctx.Request.URL.String())

			err := next(ctx)

			duration := time.Since(start)
			if err != nil {
				lm.logger.Errorf("Request failed: %s %s - Error: %v - Duration: %v",
					ctx.Request.Method, ctx.Request.URL.String(), err, duration)
			} else if ctx.Response != nil {
				lm.logger.Infof("Request: %s %s - Status: %d - Duration: %v",
					ctx.Request.Method, ctx.Request.URL.String(), ctx.Response.StatusCode, duration)
			}

			return err
		}
	}
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	token     string
	tokenType string
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(token string) Middleware {
	return NewAuthMiddlewareWithType("Bearer", token)
}

// NewAuthMiddlewareWithType 创建带类型的认证中间件
func NewAuthMiddlewareWithType(tokenType, token string) Middleware {
	am := &AuthMiddleware{
		token:     token,
		tokenType: tokenType,
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if am.token != "" {
				ctx.Request.Header.Set("Authorization", fmt.Sprintf("%s %s", am.tokenType, am.token))
			}
			return next(ctx)
		}
	}
}

// BasicAuthMiddleware Basic认证中间件
type BasicAuthMiddleware struct {
	username string
	password string
}

// NewBasicAuthMiddleware 创建Basic认证中间件
func NewBasicAuthMiddleware(username, password string) Middleware {
	bam := &BasicAuthMiddleware{
		username: username,
		password: password,
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if bam.username != "" {
				ctx.Request.SetBasicAuth(bam.username, bam.password)
			}
			return next(ctx)
		}
	}
}

// RetryMiddleware 重试中间件
type RetryMiddleware struct {
	maxRetries int
	backoff    BackoffStrategy
}

// NewRetryMiddleware 创建重试中间件
func NewRetryMiddleware(maxRetries int, backoff BackoffStrategy) Middleware {
	rm := &RetryMiddleware{
		maxRetries: maxRetries,
		backoff:    backoff,
	}
	if backoff == nil {
		rm.backoff = NewExponentialBackoff(100*time.Millisecond, 5*time.Second)
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			var lastErr error

			for retry := 0; retry <= rm.maxRetries; retry++ {
				if retry > 0 {
					// 执行退避
					backoffDuration := rm.backoff.Next(retry - 1)
					time.Sleep(backoffDuration)
				}

				err := next(ctx)
				if err == nil {
					// 成功
					if ctx.Response != nil && ctx.Response.StatusCode >= 500 && retry < rm.maxRetries {
						// 服务器错误，继续重试
						lastErr = fmt.Errorf("server error: %d", ctx.Response.StatusCode)
						continue
					}
					return nil
				}

				lastErr = err
			}

			return lastErr
		}
	}
}

// TimeoutMiddleware 超时中间件
type TimeoutMiddleware struct {
	timeout time.Duration
}

// NewTimeoutMiddleware 创建超时中间件
func NewTimeoutMiddleware(timeout time.Duration) Middleware {
	tm := &TimeoutMiddleware{
		timeout: timeout,
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if tm.timeout > 0 {
				originalCtx := ctx.Request.Context()
				timeoutCtx, timeoutCancel := context.WithTimeout(originalCtx, tm.timeout)
				ctx.Request = ctx.Request.WithContext(timeoutCtx)
				defer timeoutCancel()
			}
			return next(ctx)
		}
	}
}

// HeaderMiddleware 请求头中间件
type HeaderMiddleware struct {
	headers map[string]string
}

// NewHeaderMiddleware 创建请求头中间件
func NewHeaderMiddleware(headers map[string]string) Middleware {
	hm := &HeaderMiddleware{
		headers: headers,
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			for key, value := range hm.headers {
				ctx.Request.Header.Set(key, value)
			}
			return next(ctx)
		}
	}
}

// UserAgentMiddleware User-Agent中间件
type UserAgentMiddleware struct {
	userAgent string
}

// NewUserAgentMiddleware 创建User-Agent中间件
func NewUserAgentMiddleware(userAgent string) Middleware {
	um := &UserAgentMiddleware{
		userAgent: userAgent,
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if um.userAgent != "" {
				ctx.Request.Header.Set("User-Agent", um.userAgent)
			}
			return next(ctx)
		}
	}
}

// RequestIDMiddleware 请求ID中间件
type RequestIDMiddleware struct {
	generator func() string
}

// NewRequestIDMiddleware 创建请求ID中间件
func NewRequestIDMiddleware(generator func() string) Middleware {
	rm := &RequestIDMiddleware{
		generator: generator,
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			requestID := ""
			if rm.generator != nil {
				requestID = rm.generator()
			} else {
				requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			}

			ctx.Request.Header.Set("X-Request-ID", requestID)
			ctx.Metadata["request_id"] = requestID

			return next(ctx)
		}
	}
}

// MetricsMiddleware 指标收集中间件
type MetricsMiddleware struct {
	onRequest  func(method, url string)
	onResponse func(method, url string, statusCode int, duration time.Duration)
	onError    func(method, url string, err error)
}

// NewMetricsMiddleware 创建指标收集中间件
func NewMetricsMiddleware(
	onRequest func(method, url string),
	onResponse func(method, url string, statusCode int, duration time.Duration),
	onError func(method, url string, err error),
) Middleware {
	mm := &MetricsMiddleware{
		onRequest:  onRequest,
		onResponse: onResponse,
		onError:    onError,
	}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()

			if mm.onRequest != nil {
				mm.onRequest(ctx.Request.Method, ctx.Request.URL.String())
			}

			err := next(ctx)
			duration := time.Since(start)

			if err != nil {
				if mm.onError != nil {
					mm.onError(ctx.Request.Method, ctx.Request.URL.String(), err)
				}
			} else if mm.onResponse != nil && ctx.Response != nil {
				mm.onResponse(ctx.Request.Method, ctx.Request.URL.String(), ctx.Response.StatusCode, duration)
			}

			return err
		}
	}
}
