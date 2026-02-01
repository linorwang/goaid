package sendsms

import "errors"

var (
	ErrInvalidPhone       = errors.New("invalid phone number")
	ErrEmptyTemplate      = errors.New("template ID is empty")
	ErrEmptySignName      = errors.New("sign name is empty")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrCodeNotFound       = errors.New("verification code not found")
	ErrCodeExpired        = errors.New("verification code expired")
	ErrCodeInvalid        = errors.New("verification code invalid")
	ErrSendFailed         = errors.New("send SMS failed")
	ErrConfigInvalid      = errors.New("invalid configuration")
	ErrAllProvidersFailed = errors.New("all providers failed")
	ErrTimeout            = errors.New("request timeout")
	ErrNetworkError       = errors.New("network error")
	ErrAuthFailed         = errors.New("authentication failed")
)

// SMSError 短信错误包装
type SMSError struct {
	Code      string
	Message   string
	ErrorType ErrorType
	Retryable bool
	Provider  string
	Original  error
}

// Error 实现 error 接口
func (e *SMSError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Original != nil {
		return e.Original.Error()
	}
	return "unknown SMS error"
}

// Unwrap 实现 errors.Unwrap 接口
func (e *SMSError) Unwrap() error {
	return e.Original
}

// NewSMSError 创建短信错误
func NewSMSError(code, message string, errorType ErrorType, retryable bool, provider string, original error) *SMSError {
	return &SMSError{
		Code:      code,
		Message:   message,
		ErrorType: errorType,
		Retryable: retryable,
		Provider:  provider,
		Original:  original,
	}
}
