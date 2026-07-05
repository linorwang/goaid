package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RequestOption configures a single request.
type RequestOption func(*RequestConfig)

// RequestConfig stores request-scoped settings built from RequestOption values.
type RequestConfig struct {
	Timeout     time.Duration
	Headers     map[string]string
	QueryParams map[string]string
	Body        []byte
	MaxRetries  int

	err               error
	maxRetriesSet     bool
	basicAuthUsername string
	basicAuthPassword string
	basicAuthSet      bool
}

// Client sends HTTP requests.
//
// Use Do for common API calls where the response body should be read eagerly.
// Use Get, Post, Put, Delete, or Send when callers need the raw *http.Response.
type Client interface {
	Get(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	Post(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	Put(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	Delete(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	Send(ctx context.Context, method, url string, opts ...RequestOption) (*http.Response, error)
	Do(ctx context.Context, method, url string, opts ...RequestOption) (*Response, error)
	Use(middleware ...Middleware)
	Clone() Client
}

// ResponseWrapper is kept for backward compatibility.
type ResponseWrapper struct {
	*http.Response
	BodyBytes []byte
}

// Response wraps an HTTP response whose body has already been read.
type Response struct {
	*http.Response
	bodyBytes []byte
	readTime  time.Duration
}

// String returns the response body as a string.
func (r *Response) String() string {
	if r == nil || r.bodyBytes == nil {
		return ""
	}
	return string(r.bodyBytes)
}

// Bytes returns a copy of the response body bytes.
func (r *Response) Bytes() []byte {
	if r == nil || r.bodyBytes == nil {
		return nil
	}
	return append([]byte(nil), r.bodyBytes...)
}

// JSON decodes the response body into v.
func (r *Response) JSON(v any) error {
	if r == nil || len(r.bodyBytes) == 0 {
		return fmt.Errorf("response body is empty")
	}
	return json.Unmarshal(r.bodyBytes, v)
}

// Success reports whether the status code is in the 2xx range.
func (r *Response) Success() bool {
	return r != nil && r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices
}

// Error returns an HTTPError for non-2xx responses.
func (r *Response) Error() *HTTPError {
	if r == nil {
		return &HTTPError{Message: "response is nil"}
	}
	if r.Success() {
		return nil
	}
	return &HTTPError{
		StatusCode: r.StatusCode,
		Message:    r.String(),
		Response:   r,
	}
}

// ReadTime returns how long it took to read the response body.
func (r *Response) ReadTime() time.Duration {
	if r == nil {
		return 0
	}
	return r.readTime
}

// HTTPError describes a non-2xx HTTP response.
type HTTPError struct {
	StatusCode int
	Message    string
	Response   *Response
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return fmt.Sprintf("HTTP error: status=%d", e.StatusCode)
	}
	return fmt.Sprintf("HTTP error: status=%d, message=%s", e.StatusCode, e.Message)
}

// IsNotFound reports whether the response status is 404.
func (e *HTTPError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsTimeout reports whether the response status is 408 or 504.
func (e *HTTPError) IsTimeout() bool {
	return e.StatusCode == http.StatusRequestTimeout ||
		e.StatusCode == http.StatusGatewayTimeout
}

// IsClientError reports whether the response status is in the 4xx range.
func (e *HTTPError) IsClientError() bool {
	return e.StatusCode >= http.StatusBadRequest && e.StatusCode < http.StatusInternalServerError
}

// IsServerError reports whether the response status is in the 5xx range.
func (e *HTTPError) IsServerError() bool {
	return e.StatusCode >= http.StatusInternalServerError && e.StatusCode < 600
}

// Middleware wraps a request handler.
type Middleware func(next Handler) Handler

// Handler sends or handles a request.
type Handler func(ctx *Context) error

// Context is passed through middleware for a single request attempt.
type Context struct {
	Request   *http.Request
	Response  *http.Response
	Error     error
	Metadata  map[string]any
	StartTime time.Time
}

// NewContext creates middleware context for req.
func NewContext(req *http.Request) *Context {
	return &Context{
		Request:   req,
		Metadata:  make(map[string]any),
		StartTime: time.Now(),
	}
}

// BackoffStrategy returns the delay before a retry attempt.
type BackoffStrategy interface {
	Next(retry int) time.Duration
}

// LinearBackoff increases the delay linearly.
type LinearBackoff struct {
	Interval time.Duration
}

func (lb *LinearBackoff) Next(retry int) time.Duration {
	return time.Duration(retry+1) * lb.Interval
}

// ExponentialBackoff increases the delay exponentially up to Max.
type ExponentialBackoff struct {
	Initial time.Duration
	Max     time.Duration
}

func (eb *ExponentialBackoff) Next(retry int) time.Duration {
	duration := eb.Initial * time.Duration(1<<uint(retry))
	if duration > eb.Max {
		duration = eb.Max
	}
	return duration
}

// ConstantBackoff returns the same delay for every retry.
type ConstantBackoff struct {
	Interval time.Duration
}

func (cb *ConstantBackoff) Next(retry int) time.Duration {
	return cb.Interval
}

// NewLinearBackoff creates a linear backoff strategy.
func NewLinearBackoff(interval time.Duration) BackoffStrategy {
	return &LinearBackoff{Interval: interval}
}

// NewExponentialBackoff creates an exponential backoff strategy.
func NewExponentialBackoff(initial, max time.Duration) BackoffStrategy {
	return &ExponentialBackoff{Initial: initial, Max: max}
}

// NewConstantBackoff creates a constant backoff strategy.
func NewConstantBackoff(interval time.Duration) BackoffStrategy {
	return &ConstantBackoff{Interval: interval}
}

// ReadResponse reads and closes resp.Body, then returns a Response wrapper.
func ReadResponse(resp *http.Response) (*Response, error) {
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	startTime := time.Now()
	var bodyBytes []byte
	var err error

	if resp.Body != nil {
		bodyBytes, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	return &Response{
		Response:  resp,
		bodyBytes: bodyBytes,
		readTime:  time.Since(startTime),
	}, err
}
