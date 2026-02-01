package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RequestOption 定义请求选项
type RequestOption func(*RequestConfig)

// RequestConfig 请求配置
type RequestConfig struct {
	Timeout     time.Duration
	Headers     map[string]string
	QueryParams map[string]string
	Body        []byte
	MaxRetries  int
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

	// Do 发送请求并返回包装后的响应
	Do(ctx context.Context, method, url string, opts ...RequestOption) (*Response, error)

	// Use 添加中间件
	Use(middleware ...Middleware)

	// Clone 克隆客户端
	Clone() Client
}

// ResponseWrapper 响应包装器（保留向后兼容）
type ResponseWrapper struct {
	*http.Response
	BodyBytes []byte
}

// Response 增强的响应包装器
type Response struct {
	*http.Response
	bodyBytes []byte
	readTime  time.Duration
}

// String 返回响应体字符串
func (r *Response) String() string {
	if r.bodyBytes == nil {
		return ""
	}
	return string(r.bodyBytes)
}

// Bytes 返回响应体字节数组
func (r *Response) Bytes() []byte {
	return r.bodyBytes
}

// JSON 将响应体反序列化到指定对象
func (r *Response) JSON(v any) error {
	if r.bodyBytes == nil {
		return fmt.Errorf("response body is empty")
	}
	return json.Unmarshal(r.bodyBytes, v)
}

// Success 判断响应是否成功（2xx状态码）
func (r *Response) Success() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// Error 返回响应错误
func (r *Response) Error() error {
	if r.Success() {
		return nil
	}
	return &HTTPError{
		StatusCode: r.StatusCode,
		Message:    r.String(),
		Response:   r,
	}
}

// HTTPError HTTP错误
type HTTPError struct {
	StatusCode int
	Message    string
	Response   *Response
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: status=%d, message=%s", e.StatusCode, e.Message)
}

// IsNotFound 判断是否为404错误
func (e *HTTPError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsTimeout 判断是否为超时错误
func (e *HTTPError) IsTimeout() bool {
	return e.StatusCode == http.StatusRequestTimeout ||
		e.StatusCode == http.StatusGatewayTimeout
}

// IsClientError 判断是否为客户端错误（4xx）
func (e *HTTPError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}

// IsServerError 判断是否为服务器错误（5xx）
func (e *HTTPError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// ========== 中间件系统 ==========

// Middleware 中间件类型
type Middleware func(next Handler) Handler

// Handler 处理器类型
type Handler func(ctx *Context) error

// Context 中间件上下文
type Context struct {
	Request   *http.Request
	Response  *http.Response
	Error     error
	Metadata  map[string]any
	StartTime time.Time
}

// NewContext 创建新的中间件上下文
func NewContext(req *http.Request) *Context {
	return &Context{
		Request:   req,
		Metadata:  make(map[string]any),
		StartTime: time.Now(),
	}
}

// ========== 重试策略 ==========

// BackoffStrategy 退避策略接口
type BackoffStrategy interface {
	Next(retry int) time.Duration
}

// LinearBackoff 线性退避
type LinearBackoff struct {
	Interval time.Duration
}

func (lb *LinearBackoff) Next(retry int) time.Duration {
	return time.Duration(retry+1) * lb.Interval
}

// ExponentialBackoff 指数退避
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

// ConstantBackoff 常数退避
type ConstantBackoff struct {
	Interval time.Duration
}

func (cb *ConstantBackoff) Next(retry int) time.Duration {
	return cb.Interval
}

// NewLinearBackoff 创建线性退避策略
func NewLinearBackoff(interval time.Duration) BackoffStrategy {
	return &LinearBackoff{Interval: interval}
}

// NewExponentialBackoff 创建指数退避策略
func NewExponentialBackoff(initial, max time.Duration) BackoffStrategy {
	return &ExponentialBackoff{Initial: initial, Max: max}
}

// NewConstantBackoff 创建常数退避策略
func NewConstantBackoff(interval time.Duration) BackoffStrategy {
	return &ConstantBackoff{Interval: interval}
}

// ReadResponse 读取响应体并返回包装后的响应
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
	}

	return &Response{
		Response:  resp,
		bodyBytes: bodyBytes,
		readTime:  time.Since(startTime),
	}, err
}
