package sendsms

import (
	"context"
	"math"
	"time"
)

// RetryManager 重试管理器
type RetryManager struct {
	config *Config
}

// NewRetryManager 创建重试管理器
func NewRetryManager(config *Config) *RetryManager {
	if config == nil {
		config = DefaultConfig()
	}
	return &RetryManager{
		config: config,
	}
}

// Retry 带重试的发送
func (r *RetryManager) Retry(ctx context.Context, fn func() (*SMSResponse, error)) (*SMSResponse, error) {
	var lastResp *SMSResponse
	var lastErr error

	for attempt := 0; attempt <= r.config.RetryTimes; attempt++ {
		if attempt > 0 {
			// 计算延迟
			delay := r.GetDelay(attempt)

			// 可取消的睡眠
			if err := r.SleepWithCancel(ctx, delay); err != nil {
				return nil, err
			}
		}

		resp, err := fn()
		if err == nil {
			// 成功，返回结果
			if resp != nil {
				resp.RetryCount = attempt
			}
			return resp, nil
		}

		// 记录最后一次错误
		lastResp = resp
		lastErr = err

		// 检查是否应该重试
		if !r.ShouldRetry(err, attempt) {
			break
		}
	}

	// 所有重试都失败
	if lastResp != nil {
		lastResp.RetryCount = r.config.RetryTimes
	}
	return lastResp, lastErr
}

// GetDelay 获取重试延迟
func (r *RetryManager) GetDelay(attempt int) time.Duration {
	var delay time.Duration

	switch r.config.RetryStrategy {
	case RetryFixedDelay:
		delay = r.config.RetryDelay

	case RetryExponentialBackoff:
		// 指数退避: delay = min(initialDelay * (2^attempt), maxDelay)
		multiplier := math.Pow(2, float64(attempt))
		delay = time.Duration(float64(r.config.RetryDelay) * multiplier)

	case RetryLinearBackoff:
		// 线性退避: delay = min(initialDelay + (multiplier * attempt), maxDelay)
		delay = r.config.RetryDelay + time.Duration(r.config.RetryMultiplier*float64(attempt)*float64(time.Second))
	}

	// 限制最大延迟
	if delay > r.config.MaxRetryDelay {
		delay = r.config.MaxRetryDelay
	}

	return delay
}

// ShouldRetry 判断是否应该重试
func (r *RetryManager) ShouldRetry(err error, attempt int) bool {
	// 达到最大重试次数
	if attempt >= r.config.RetryTimes {
		return false
	}

	// 检查错误类型
	if smsErr, ok := err.(*SMSError); ok {
		return smsErr.Retryable
	}

	// 对于未知错误，默认可重试
	return true
}

// SleepWithCancel 可取消的睡眠
func (r *RetryManager) SleepWithCancel(ctx context.Context, delay time.Duration) error {
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
