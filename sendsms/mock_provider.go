package sendsms

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// MockProvider 模拟服务商（用于测试）
type MockProvider struct {
	name     string
	failRate float64 // 失度率 0.0-1.0
	latency  time.Duration
}

// NewMockProvider 创建模拟服务商
func NewMockProvider(name string, failRate float64, latency time.Duration) *MockProvider {
	return &MockProvider{
		name:     name,
		failRate: failRate,
		latency:  latency,
	}
}

// Send 发送短信
func (m *MockProvider) Send(ctx context.Context, req *SMSRequest) (*SMSResponse, error) {
	// 模拟延迟
	if m.latency > 0 {
		time.Sleep(m.latency)
	}

	// 模拟失败
	if rand.Float64() < m.failRate {
		return &SMSResponse{
			Success:  false,
			Provider: m.name,
			Message:  "mock send failed",
		}, NewSMSError("MOCK_ERROR", "mock send failed", ErrorTypeProvider, true, m.name, ErrSendFailed)
	}

	return &SMSResponse{
		Success:   true,
		Provider:  m.name,
		Message:   "mock send success",
		MessageID: fmt.Sprintf("mock_%d", time.Now().UnixNano()),
		RequestID: fmt.Sprintf("req_%d", time.Now().UnixNano()),
	}, nil
}

// SendBatch 批量发送
func (m *MockProvider) SendBatch(ctx context.Context, reqs []*SMSRequest) ([]*SMSResponse, error) {
	responses := make([]*SMSResponse, len(reqs))

	for i, req := range reqs {
		resp, err := m.Send(ctx, req)
		responses[i] = resp
		if err != nil {
			return responses, err
		}
	}

	return responses, nil
}

// Name 获取服务商名称
func (m *MockProvider) Name() string {
	return m.name
}

// ValidateConfig 验证配置
func (m *MockProvider) ValidateConfig() error {
	return nil
}

// GetBalance 获取余额（模拟）
func (m *MockProvider) GetBalance(ctx context.Context) (*Balance, error) {
	return &Balance{
		Amount:    1000.0,
		Currency:  "CNY",
		UpdatedAt: time.Now(),
	}, nil
}

// HealthCheck 健康检查
func (m *MockProvider) HealthCheck(ctx context.Context) bool {
	return true
}

// GetErrorType 获取错误类型
func (m *MockProvider) GetErrorType(err error) ErrorType {
	if smsErr, ok := err.(*SMSError); ok {
		return smsErr.ErrorType
	}
	return ErrorTypeInvalid
}

// IsRetryable 判断是否可重试
func (m *MockProvider) IsRetryable(err error) bool {
	if smsErr, ok := err.(*SMSError); ok {
		return smsErr.Retryable
	}
	return true
}
