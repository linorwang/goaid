package httpclient

import (
	"context"
	"fmt"
	"log"
	"time"
)

// LoggerMiddleware logs request lifecycle events.
type LoggerMiddleware struct {
	logger Logger
}

// Logger is the minimal logging interface used by LoggerMiddleware.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

// DefaultLogger logs through the standard library log package.
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

// NewLoggerMiddleware creates logging middleware.
func NewLoggerMiddleware(logger Logger) Middleware {
	lm := &LoggerMiddleware{logger: logger}
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

// AuthMiddleware adds an Authorization header.
type AuthMiddleware struct {
	token     string
	tokenType string
}

// NewAuthMiddleware creates Bearer token middleware.
func NewAuthMiddleware(token string) Middleware {
	return NewAuthMiddlewareWithType("Bearer", token)
}

// NewAuthMiddlewareWithType creates token middleware with a custom token type.
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

// BasicAuthMiddleware adds HTTP Basic Auth.
type BasicAuthMiddleware struct {
	username string
	password string
}

// NewBasicAuthMiddleware creates Basic Auth middleware.
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

// RetryMiddleware retries failed requests inside the middleware chain.
type RetryMiddleware struct {
	maxRetries int
	backoff    BackoffStrategy
}

// NewRetryMiddleware creates retry middleware.
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
			if rm.maxRetries < 0 {
				return next(ctx)
			}

			var lastErr error
			for attempt := 0; attempt <= rm.maxRetries; attempt++ {
				if attempt > 0 {
					if err := sleepWithContext(ctx.Request.Context(), rm.backoff.Next(attempt-1)); err != nil {
						return err
					}
					if err := resetBodyForRetry(ctx.Request); err != nil {
						return err
					}
					ctx.Response = nil
					ctx.Error = nil
				}

				err := next(ctx)
				if err == nil {
					if shouldRetryResponse(ctx.Response, attempt, rm.maxRetries) {
						lastErr = fmt.Errorf("server error: %d", ctx.Response.StatusCode)
						drainAndClose(ctx.Response.Body)
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

// TimeoutMiddleware adds a per-request timeout.
type TimeoutMiddleware struct {
	timeout time.Duration
}

// NewTimeoutMiddleware creates timeout middleware.
func NewTimeoutMiddleware(timeout time.Duration) Middleware {
	tm := &TimeoutMiddleware{timeout: timeout}

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

// HeaderMiddleware adds static request headers.
type HeaderMiddleware struct {
	headers map[string]string
}

// NewHeaderMiddleware creates static header middleware.
func NewHeaderMiddleware(headers map[string]string) Middleware {
	hm := &HeaderMiddleware{headers: headers}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			for key, value := range hm.headers {
				ctx.Request.Header.Set(key, value)
			}
			return next(ctx)
		}
	}
}

// UserAgentMiddleware adds a User-Agent header.
type UserAgentMiddleware struct {
	userAgent string
}

// NewUserAgentMiddleware creates User-Agent middleware.
func NewUserAgentMiddleware(userAgent string) Middleware {
	um := &UserAgentMiddleware{userAgent: userAgent}

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if um.userAgent != "" {
				ctx.Request.Header.Set("User-Agent", um.userAgent)
			}
			return next(ctx)
		}
	}
}

// RequestIDMiddleware adds an X-Request-ID header.
type RequestIDMiddleware struct {
	generator func() string
}

// NewRequestIDMiddleware creates request ID middleware.
func NewRequestIDMiddleware(generator func() string) Middleware {
	rm := &RequestIDMiddleware{generator: generator}

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

// MetricsMiddleware emits request metrics through callbacks.
type MetricsMiddleware struct {
	onRequest  func(method, url string)
	onResponse func(method, url string, statusCode int, duration time.Duration)
	onError    func(method, url string, err error)
}

// NewMetricsMiddleware creates metrics middleware.
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
