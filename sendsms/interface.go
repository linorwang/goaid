package sendsms

import "context"

// SMSProvider 服务商接口
type SMSProvider interface {
	// 发送短信（带容错和重试）
	Send(ctx context.Context, req *SMSRequest) (*SMSResponse, error)

	// 批量发送（带容错）
	SendBatch(ctx context.Context, reqs []*SMSRequest) ([]*SMSResponse, error)

	// 获取服务商名称
	Name() string

	// 检查配置是否有效
	ValidateConfig() error

	// 获取余额（如果支持）
	GetBalance(ctx context.Context) (*Balance, error)

	// 健康检查
	HealthCheck(ctx context.Context) bool

	// 获取错误类型
	GetErrorType(err error) ErrorType

	// 判断错误是否可重试
	IsRetryable(err error) bool
}

// FailoverManager Failover 管理接口
type FailoverManager interface {
	// 获取可用服务商
	GetAvailableProvider() SMSProvider

	// 标记服务商失败
	MarkProviderFailed(provider string)

	// 标记服务商恢复健康
	MarkProviderHealthy(provider string)

	// 获取所有服务商健康状态
	GetHealthStatus() []*ProviderHealth
}
