package sendsms

import (
	"math/rand"
	"sync"
	"time"
)

// FailoverManagerImpl Failover 管理器实现
type FailoverManagerImpl struct {
	providers    map[string]SMSProvider
	primary      string
	backups      []string
	strategy     FailoverStrategy
	healthStatus map[string]*ProviderHealth
	cooldown     time.Duration
	mu           sync.RWMutex
	currentIndex int // 轮询索引
}

// NewFailoverManager 创建 Failover 管理器
func NewFailoverManager(primary string, backups []string, providers map[string]SMSProvider, strategy FailoverStrategy, cooldown time.Duration) *FailoverManagerImpl {
	healthStatus := make(map[string]*ProviderHealth)

	// 初始化所有服务商的健康状态
	if p, ok := providers[primary]; ok {
		healthStatus[primary] = &ProviderHealth{
			Name:          p.Name(),
			IsHealthy:     true,
			LastCheckTime: time.Now(),
		}
	}

	for _, backup := range backups {
		if p, ok := providers[backup]; ok {
			healthStatus[backup] = &ProviderHealth{
				Name:          p.Name(),
				IsHealthy:     true,
				LastCheckTime: time.Now(),
			}
		}
	}

	return &FailoverManagerImpl{
		providers:    providers,
		primary:      primary,
		backups:      backups,
		strategy:     strategy,
		healthStatus: healthStatus,
		cooldown:     cooldown,
		currentIndex: 0,
	}
}

// GetAvailableProvider 获取可用服务商
func (f *FailoverManagerImpl) GetAvailableProvider() SMSProvider {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch f.strategy {
	case FailoverSequential:
		return f.getSequentialProvider()
	case FailoverRandom:
		return f.getRandomProvider()
	case FailoverRoundRobin:
		return f.getRoundRobinProvider()
	default:
		return f.getSequentialProvider()
	}
}

// MarkProviderFailed 标记服务商失败
func (f *FailoverManagerImpl) MarkProviderFailed(provider string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if health, ok := f.healthStatus[provider]; ok {
		health.IsHealthy = false
		health.ErrorCount++
		health.LastErrorTime = time.Now()
	}
}

// MarkProviderHealthy 标记服务商恢复健康
func (f *FailoverManagerImpl) MarkProviderHealthy(provider string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if health, ok := f.healthStatus[provider]; ok {
		health.IsHealthy = true
		health.LastCheckTime = time.Now()
	}
}

// GetHealthStatus 获取健康状态
func (f *FailoverManagerImpl) GetHealthStatus() []*ProviderHealth {
	f.mu.RLock()
	defer f.mu.RUnlock()

	status := make([]*ProviderHealth, 0, len(f.healthStatus))
	for _, health := range f.healthStatus {
		status = append(status, health)
	}
	return status
}

// isInCooldown 判断是否在冷却期
func (f *FailoverManagerImpl) isInCooldown(provider string) bool {
	health, ok := f.healthStatus[provider]
	if !ok || health.IsHealthy {
		return false
	}

	return time.Since(health.LastErrorTime) < f.cooldown
}

// getSequentialProvider 顺序获取服务商
func (f *FailoverManagerImpl) getSequentialProvider() SMSProvider {
	// 先尝试主服务商
	if p, ok := f.providers[f.primary]; ok {
		if f.isInCooldown(f.primary) {
			// 主服务商在冷却期，尝试备用
		} else {
			return p
		}
	}

	// 尝试备用服务商
	for _, backup := range f.backups {
		if p, ok := f.providers[backup]; ok {
			if !f.isInCooldown(backup) {
				return p
			}
		}
	}

	// 所有服务商都不可用，返回主服务商让调用者处理
	if p, ok := f.providers[f.primary]; ok {
		return p
	}

	return nil
}

// getRandomProvider 随机获取服务商
func (f *FailoverManagerImpl) getRandomProvider() SMSProvider {
	available := make([]string, 0)

	// 收集可用的服务商
	if !f.isInCooldown(f.primary) {
		if _, ok := f.providers[f.primary]; ok {
			available = append(available, f.primary)
		}
	}

	for _, backup := range f.backups {
		if !f.isInCooldown(backup) {
			if _, ok := f.providers[backup]; ok {
				available = append(available, backup)
			}
		}
	}

	// 如果没有可用的，返回主服务商
	if len(available) == 0 {
		if p, ok := f.providers[f.primary]; ok {
			return p
		}
		return nil
	}

	// 随机选择
	index := rand.Intn(len(available))
	return f.providers[available[index]]
}

// getRoundRobinProvider 轮询获取服务商
func (f *FailoverManagerImpl) getRoundRobinProvider() SMSProvider {
	allProviders := make([]string, 0, 1+len(f.backups))

	// 构建所有服务商列表
	allProviders = append(allProviders, f.primary)
	allProviders = append(allProviders, f.backups...)

	// 找到下一个可用的服务商
	for i := 0; i < len(allProviders); i++ {
		provider := allProviders[f.currentIndex]
		f.currentIndex = (f.currentIndex + 1) % len(allProviders)

		if !f.isInCooldown(provider) {
			if p, ok := f.providers[provider]; ok {
				return p
			}
		}
	}

	// 所有服务商都在冷却期，返回主服务商
	if p, ok := f.providers[f.primary]; ok {
		return p
	}

	return nil
}
