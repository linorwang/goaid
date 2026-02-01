package sendsms

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// SMSClient 短信客户端
type SMSClient struct {
	provider    SMSProvider
	failoverMgr *FailoverManagerImpl
	retryMgr    *RetryManager
	cache       *SMSCache
	config      *Config
	providers   map[string]SMSProvider
	mu          sync.RWMutex
}

// NewSMSClient 创建短信客户端
func NewSMSClient(primary string, backups []string, providers map[string]SMSProvider, cache redis.Cmdable, config *Config) (*SMSClient, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 验证主服务商存在
	if _, ok := providers[primary]; !ok {
		return nil, fmt.Errorf("primary provider %s not found", primary)
	}

	client := &SMSClient{
		providers: providers,
		config:    config,
		cache:     NewSMSCache(cache, config.CacheConfig),
		retryMgr:  NewRetryManager(config),
	}

	// 初始化 Failover 管理器
	if config.EnableFailover {
		client.failoverMgr = NewFailoverManager(
			primary,
			backups,
			providers,
			config.FailoverStrategy,
			config.FailoverCooldown,
		)
	} else {
		// 不启用 Failover，只使用主服务商
		client.provider = providers[primary]
	}

	return client, nil
}

// Send 发送短信（带容错和 Failover）
func (c *SMSClient) Send(ctx context.Context, req *SMSRequest) (*SMSResponse, error) {
	// 验证请求
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// 限流检查
	if c.cache != nil && c.config.CacheConfig.EnableLimit {
		canSend, err := c.cache.CheckLimit(ctx, req.Phone)
		if err != nil {
			return nil, fmt.Errorf("check limit failed: %w", err)
		}
		if !canSend {
			return &SMSResponse{
				Success: false,
				Message: "rate limit exceeded",
				Error:   ErrRateLimitExceeded,
			}, ErrRateLimitExceeded
		}
	}

	// 记录发送尝试
	if c.cache != nil {
		_ = c.cache.RecordAttempt(ctx, req.Phone)
	}

	var provider SMSProvider
	var lastProviderName string
	var resp *SMSResponse
	var err error

	// 如果启用了 Failover
	if c.failoverMgr != nil {
		provider = c.failoverMgr.GetAvailableProvider()
		lastProviderName = provider.Name()

		// 使用重试管理器发送
		resp, err = c.retryMgr.Retry(ctx, func() (*SMSResponse, error) {
			return provider.Send(ctx, req)
		})

		if err != nil {
			// 标记服务商失败
			c.failoverMgr.MarkProviderFailed(lastProviderName)

			// 尝试 Failover
			backupProvider := c.failoverMgr.GetAvailableProvider()
			if backupProvider != nil && backupProvider.Name() != lastProviderName {
				resp, err = c.retryMgr.Retry(ctx, func() (*SMSResponse, error) {
					backupResp, backupErr := backupProvider.Send(ctx, req)
					if backupErr == nil {
						// 成功，记录 Failover
						if c.cache != nil {
							_ = c.cache.SaveFailoverRecord(ctx, req.Phone, lastProviderName, backupProvider.Name())
						}
						c.failoverMgr.MarkProviderHealthy(backupProvider.Name())
					}
					return backupResp, backupErr
				})
			}
		} else {
			// 成功，标记服务商健康
			c.failoverMgr.MarkProviderHealthy(lastProviderName)
		}
	} else {
		// 不启用 Failover，直接发送
		resp, err = c.retryMgr.Retry(ctx, func() (*SMSResponse, error) {
			return c.provider.Send(ctx, req)
		})
	}

	// 设置响应信息
	if resp != nil {
		resp.Provider = lastProviderName
		resp.Duration = time.Since(startTime)
	}

	return resp, err
}

// SendBatch 批量发送（带容错）
func (c *SMSClient) SendBatch(ctx context.Context, reqs []*SMSRequest) (*BatchResult, error) {
	if len(reqs) == 0 {
		return &BatchResult{
			Total:     0,
			Success:   0,
			Failed:    0,
			Responses: []*SMSResponse{},
		}, nil
	}

	result := &BatchResult{
		Total:      len(reqs),
		Responses:  make([]*SMSResponse, len(reqs)),
		FailedReqs: []*SMSRequest{},
	}

	// 逐个发送
	for i, req := range reqs {
		resp, err := c.Send(ctx, req)
		result.Responses[i] = resp

		if err != nil || (resp != nil && !resp.Success) {
			result.Failed++
			result.FailedReqs = append(result.FailedReqs, req)
		} else {
			result.Success++
		}
	}

	return result, nil
}

// SendBatchConcurrent 并发批量发送
func (c *SMSClient) SendBatchConcurrent(ctx context.Context, reqs []*SMSRequest, concurrency int) (*BatchResult, error) {
	if len(reqs) == 0 {
		return &BatchResult{
			Total:     0,
			Success:   0,
			Failed:    0,
			Responses: []*SMSResponse{},
		}, nil
	}

	if concurrency <= 0 {
		concurrency = c.config.ConcurrentLimit
	}
	if concurrency <= 0 {
		concurrency = 10
	}

	result := &BatchResult{
		Total:      len(reqs),
		Responses:  make([]*SMSResponse, len(reqs)),
		FailedReqs: []*SMSRequest{},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, concurrency)

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *SMSRequest) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			resp, err := c.Send(ctx, r)

			mu.Lock()
			result.Responses[idx] = resp
			if err != nil || (resp != nil && !resp.Success) {
				result.Failed++
				result.FailedReqs = append(result.FailedReqs, r)
			} else {
				result.Success++
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return result, nil
}

// SendVerificationCode 发送验证码
func (c *SMSClient) SendVerificationCode(ctx context.Context, req *VerificationCodeRequest) (*SMSResponse, error) {
	// 检查频率限制
	if c.cache != nil {
		canSend, err := c.cache.CheckLimit(ctx, req.Phone)
		if err != nil {
			return nil, fmt.Errorf("check limit failed: %w", err)
		}
		if !canSend {
			return &SMSResponse{
				Success: false,
				Message: "rate limit exceeded",
				Error:   ErrRateLimitExceeded,
			}, ErrRateLimitExceeded
		}
	}

	// 生成验证码
	code, err := c.generateVerificationCode(req.CodeLength)
	if err != nil {
		return nil, fmt.Errorf("generate verification code failed: %w", err)
	}

	// 保存验证码
	if c.cache != nil {
		err = c.cache.SaveCode(ctx, req.Phone, code, req.ExpireTime)
		if err != nil {
			return nil, fmt.Errorf("save verification code failed: %w", err)
		}
	}

	// 发送短信
	smsReq := &SMSRequest{
		Phone:    req.Phone,
		Template: req.Template,
		Params:   []string{code},
		SignName: req.SignName,
		Type:     SMSVerification,
	}

	return c.Send(ctx, smsReq)
}

// VerifyCode 验证验证码
func (c *SMSClient) VerifyCode(ctx context.Context, req *VerifyCodeRequest) (*VerifyResult, error) {
	if c.cache == nil {
		return &VerifyResult{
			Valid:   false,
			Message: "cache not configured",
		}, nil
	}

	// 获取保存的验证码
	savedCode, err := c.cache.GetCode(ctx, req.Phone)
	if err != nil {
		if err == redis.Nil {
			return &VerifyResult{
				Valid:   false,
				Message: "verification code not found or expired",
			}, nil
		}
		return nil, fmt.Errorf("get verification code: %w", err)
	}

	if savedCode == "" {
		return &VerifyResult{
			Valid:   false,
			Message: "verification code not found or expired",
		}, nil
	}

	// 验证码不匹配
	if savedCode != req.Code {
		return &VerifyResult{
			Valid:   false,
			Message: "verification code invalid",
		}, nil
	}

	// 验证成功，删除验证码
	if req.CleanOnce {
		_ = c.cache.DeleteCode(ctx, req.Phone)
	}

	return &VerifyResult{
		Valid:   true,
		Message: "verification successful",
	}, nil
}

// SendWithTemplate 使用模板发送（简化接口）
func (c *SMSClient) SendWithTemplate(ctx context.Context, phone, templateID, signName string, params []string) (*SMSResponse, error) {
	return c.Send(ctx, &SMSRequest{
		Phone:    phone,
		Template: templateID,
		Params:   params,
		SignName: signName,
	})
}

// GetHealthStatus 获取所有服务商健康状态
func (c *SMSClient) GetHealthStatus() []*ProviderHealth {
	if c.failoverMgr != nil {
		return c.failoverMgr.GetHealthStatus()
	}
	return []*ProviderHealth{}
}

// RetryFailed 重试失败的请求
func (c *SMSClient) RetryFailed(ctx context.Context, failedReqs []*SMSRequest) (*BatchResult, error) {
	return c.SendBatch(ctx, failedReqs)
}

// validateRequest 验证请求
func (c *SMSClient) validateRequest(req *SMSRequest) error {
	if req.Phone == "" {
		return ErrInvalidPhone
	}
	if req.Template == "" && req.Content == "" {
		return ErrEmptyTemplate
	}
	return nil
}

// generateVerificationCode 生成验证码
func (c *SMSClient) generateVerificationCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}

	const digits = "0123456789"
	code := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}
