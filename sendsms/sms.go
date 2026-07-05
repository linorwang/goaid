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

// SMSClient is the high-level SMS sender.
type SMSClient struct {
	provider    SMSProvider
	failoverMgr *FailoverManagerImpl
	retryMgr    *RetryManager
	cache       *SMSCache
	config      *Config
	providers   map[string]SMSProvider
	mu          sync.RWMutex
}

// NewSMSClient creates an SMS client. Prefer New with ClientOption for new code.
func NewSMSClient(primary string, backups []string, providers map[string]SMSProvider, cache redis.Cmdable, config *Config) (*SMSClient, error) {
	config = normalizeConfig(config)
	if primary == "" {
		primary = config.PrimaryProvider
	}
	if len(backups) == 0 {
		backups = config.BackupProviders
	}
	if providers == nil {
		return nil, fmt.Errorf("%w: providers is nil", ErrConfigInvalid)
	}
	if primary == "" {
		return nil, fmt.Errorf("%w: primary provider is empty", ErrConfigInvalid)
	}
	if _, ok := providers[primary]; !ok {
		return nil, fmt.Errorf("primary provider %s not found", primary)
	}

	client := &SMSClient{
		providers: providers,
		config:    config,
		cache:     NewSMSCache(cache, config.CacheConfig),
		retryMgr:  NewRetryManager(config),
	}

	if config.EnableFailover {
		client.failoverMgr = NewFailoverManager(
			primary,
			backups,
			providers,
			config.FailoverStrategy,
			config.FailoverCooldown,
		)
	} else {
		client.provider = providers[primary]
	}

	return client, nil
}

// Send sends one SMS with retry, optional rate limiting, and optional failover.
func (c *SMSClient) Send(ctx context.Context, req *SMSRequest) (*SMSResponse, error) {
	ctx, cancel := c.contextWithTimeout(ctx)
	defer cancel()

	req = c.prepareRequest(req)
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	startTime := time.Now()

	if c.cache != nil && c.config.CacheConfig != nil && c.config.CacheConfig.EnableLimit {
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

	if c.cache != nil {
		_ = c.cache.RecordAttempt(ctx, req.Phone)
	}

	var resp *SMSResponse
	var err error
	var successProvider string

	if c.failoverMgr != nil {
		var firstFailedProvider string
		for _, provider := range c.failoverMgr.GetProviderCandidates() {
			if provider == nil {
				continue
			}
			providerName := provider.Name()
			resp, err = c.retryMgr.Retry(ctx, func() (*SMSResponse, error) {
				return provider.Send(ctx, req)
			})
			if err == nil {
				successProvider = providerName
				c.failoverMgr.MarkProviderHealthy(providerName)
				if firstFailedProvider != "" && c.cache != nil {
					_ = c.cache.SaveFailoverRecord(ctx, req.Phone, firstFailedProvider, providerName)
				}
				break
			}
			if firstFailedProvider == "" {
				firstFailedProvider = providerName
			}
			c.failoverMgr.MarkProviderFailed(providerName)
		}
	} else {
		resp, err = c.retryMgr.Retry(ctx, func() (*SMSResponse, error) {
			return c.provider.Send(ctx, req)
		})
		if c.provider != nil {
			successProvider = c.provider.Name()
		}
	}

	if resp != nil {
		if successProvider != "" {
			resp.Provider = successProvider
		}
		resp.Duration = time.Since(startTime)
	}

	return resp, err
}

// SendBatch sends requests one by one.
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

// SendBatchConcurrent sends requests concurrently with a bounded worker count.
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

// SendVerificationCode generates and sends a verification code.
func (c *SMSClient) SendVerificationCode(ctx context.Context, req *VerificationCodeRequest) (*SMSResponse, error) {
	if req == nil || req.Phone == "" {
		return nil, ErrInvalidPhone
	}

	code, err := c.generateVerificationCode(req.CodeLength)
	if err != nil {
		return nil, fmt.Errorf("generate verification code failed: %w", err)
	}

	if c.cache != nil {
		err = c.cache.SaveCode(ctx, req.Phone, code, req.ExpireTime)
		if err != nil {
			return nil, fmt.Errorf("save verification code failed: %w", err)
		}
	}

	smsReq := &SMSRequest{
		Phone:    req.Phone,
		Template: req.Template,
		Params:   []string{code},
		SignName: req.SignName,
		Type:     SMSVerification,
	}

	return c.Send(ctx, smsReq)
}

// VerifyCode verifies a previously saved verification code.
func (c *SMSClient) VerifyCode(ctx context.Context, req *VerifyCodeRequest) (*VerifyResult, error) {
	if req == nil || req.Phone == "" {
		return &VerifyResult{
			Valid:   false,
			Message: "invalid phone number",
		}, nil
	}
	if c.cache == nil {
		return &VerifyResult{
			Valid:   false,
			Message: "cache not configured",
		}, nil
	}

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

	if savedCode != req.Code {
		return &VerifyResult{
			Valid:   false,
			Message: "verification code invalid",
		}, nil
	}

	if req.CleanOnce {
		_ = c.cache.DeleteCode(ctx, req.Phone)
	}

	return &VerifyResult{
		Valid:   true,
		Message: "verification successful",
	}, nil
}

// SendWithTemplate sends an SMS by template.
func (c *SMSClient) SendWithTemplate(ctx context.Context, phone, templateID, signName string, params []string) (*SMSResponse, error) {
	return c.Send(ctx, &SMSRequest{
		Phone:    phone,
		Template: templateID,
		Params:   params,
		SignName: signName,
	})
}

// GetHealthStatus returns all provider health states.
func (c *SMSClient) GetHealthStatus() []*ProviderHealth {
	if c.failoverMgr != nil {
		return c.failoverMgr.GetHealthStatus()
	}
	return []*ProviderHealth{}
}

// RetryFailed retries failed requests.
func (c *SMSClient) RetryFailed(ctx context.Context, failedReqs []*SMSRequest) (*BatchResult, error) {
	return c.SendBatch(ctx, failedReqs)
}

func (c *SMSClient) validateRequest(req *SMSRequest) error {
	if req == nil || req.Phone == "" {
		return ErrInvalidPhone
	}
	if req.Template == "" && req.Content == "" {
		return ErrEmptyTemplate
	}
	return nil
}

func (c *SMSClient) prepareRequest(req *SMSRequest) *SMSRequest {
	if req == nil {
		return nil
	}
	prepared := *req
	if prepared.Template == "" {
		prepared.Template = c.config.DefaultTemplate
	}
	if prepared.SignName == "" {
		prepared.SignName = c.config.DefaultSign
	}
	return &prepared
}

func (c *SMSClient) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.config == nil || c.config.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.config.Timeout)
}

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
