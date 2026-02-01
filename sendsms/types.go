package sendsms

import "time"

// SMSType 短信类型
type SMSType int

const (
	SMSVerification SMSType = iota // 验证码
	SMSNotification                // 通知类
	SMSMarketing                   // 营销类
)

// SMSRequest 短信请求
type SMSRequest struct {
	Phone      string   // 手机号
	Template   string   // 模板ID
	Content    string   // 短信内容（部分服务商）
	Params     []string // 模板参数
	SignName   string   // 签名
	Type       SMSType  // 短信类型
	ExtID      string   // 扩展ID（用于回调匹配）
	RetryCount int      // 内部重试次数（内部使用）
}

// SMSResponse.短信响应
type SMSResponse struct {
	Success    bool          // 是否成功
	MessageID  string        // 消息ID
	Message    string        // 返回消息
	Code       string        // 状态码
	Cost       float64       // 花费
	RequestID  string        // 请求ID
	Provider   string        // 服务商名称
	RetryCount int           // 实际重试次数
	Duration   time.Duration // 耗时
	Error      error         // 错误信息
}

// VerificationCodeRequest 验证码请求
type VerificationCodeRequest struct {
	Phone      string        // 手机号
	ExpireTime time.Duration // 过期时间
	CodeLength int           // 验证码长度
	Template   string        // 模板ID
	SignName   string        // 签名
}

// VerifyCodeRequest 验证码验证请求
type VerifyCodeRequest struct {
	Phone     string // 手机号
	Code      string // 验证码
	CleanOnce bool   // 验证后是否删除
}

// VerifyResult 验证结果
type VerifyResult struct {
	Valid   bool
	Message string
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Prefix          string        // Redis key 前缀
	ExpireTime      time.Duration // 默认过期时间
	VerificationExp time.Duration // 验证码过期时间
	EnableLimit     bool          // 是否启用限流
	LimitCount      int           // 限流次数
	LimitWindow     time.Duration // 限流时间窗口
}

// BatchResult 批量发送结果
type BatchResult struct {
	Total      int            // 总数
	Success    int            // 成功数
	Failed     int            // 失败数
	Responses  []*SMSResponse // 所有响应
	FailedReqs []*SMSRequest  // 失败的请求（可重试）
}

// ProviderHealth 服务商健康状态
type ProviderHealth struct {
	Name          string
	IsHealthy     bool
	ErrorCount    int
	LastErrorTime time.Time
	LastCheckTime time.Time
	FailoverCount int
}

// RetryStrategy 重试策略类型
type RetryStrategy int

const (
	RetryFixedDelay         RetryStrategy = iota // 固定延迟
	RetryExponentialBackoff                      // 指数退避
	RetryLinearBackoff                           // 线性退避
)

// FailoverStrategy Failover 策略类型
type FailoverStrategy int

const (
	FailoverSequential FailoverStrategy = iota // 顺序切换
	FailoverRandom                             // 随机选择
	FailoverRoundRobin                         // 轮询
)

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeNetwork   ErrorType = iota // 网络错误
	ErrorTypeTimeout                    // 超时错误
	ErrorTypeProvider                   // 服务商错误
	ErrorTypeRateLimit                  // 限流错误
	ErrorTypeAuth                       // 认证错误
	ErrorTypeInvalid                    // 请求无效
)

// Balance 余额信息
type Balance struct {
	Amount    float64
	Currency  string
	UpdatedAt time.Time
}
