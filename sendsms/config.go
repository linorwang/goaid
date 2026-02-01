package sendsms

import "time"

// Config 总配置
type Config struct {
	// 服务商配置
	PrimaryProvider string   // 主服务商
	BackupProviders []string // 备用服务商列表

	// 默认设置
	DefaultSign     string // 默认签名
	DefaultTemplate string // 默认模板ID

	// 缓存配置
	CacheConfig *CacheConfig

	// 重试配置
	RetryStrategy   RetryStrategy // 重试策略
	RetryTimes      int           // 重试次数
	RetryDelay      time.Duration // 初始重试延迟
	MaxRetryDelay   time.Duration // 最大重试延迟
	RetryMultiplier float64       // 退避乘数（指数/线性）

	// Failover 配置
	EnableFailover      bool             // 是否启用 Failover
	FailoverStrategy    FailoverStrategy // Failover 策略
	FailoverCooldown    time.Duration    // Failover 冷却时间
	HealthCheckInterval time.Duration    // 健康检查间隔

	// 请求配置
	Timeout         time.Duration // 请求超时
	BatchSize       int           // 批量发送大小
	ConcurrentLimit int           // 并发限制

	// 熔断器配置
	EnableCircuitBreaker    bool          // 是否启用熔断器
	CircuitBreakerThreshold int           // 熔断阈值
	CircuitBreakerTimeout   time.Duration // 熔断超时
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		CacheConfig: &CacheConfig{
			Prefix:          "sms:",
			ExpireTime:      24 * time.Hour,
			VerificationExp: 5 * time.Minute,
			EnableLimit:     true,
			LimitCount:      5,
			LimitWindow:     time.Hour,
		},
		RetryStrategy:       RetryExponentialBackoff,
		RetryTimes:          3,
		RetryDelay:          1 * time.Second,
		MaxRetryDelay:       10 * time.Second,
		RetryMultiplier:     2.0,
		EnableFailover:      true,
		FailoverStrategy:    FailoverSequential,
		FailoverCooldown:    5 * time.Minute,
		HealthCheckInterval: 1 * time.Minute,
		Timeout:             10 * time.Second,
		BatchSize:           100,
		ConcurrentLimit:     10,
	}
}
