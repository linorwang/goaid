package sendsms

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// SMSCache Redis 缓存管理
type SMSCache struct {
	client redis.Cmdable // 接收 redis.Client 或 redis.Cmdable
	config *CacheConfig
}

// NewSMSCache 创建缓存实例
func NewSMSCache(client redis.Cmdable, config *CacheConfig) *SMSCache {
	if config == nil {
		config = &CacheConfig{}
	}
	if config.Prefix == "" {
		config.Prefix = "sms:"
	}
	if config.VerificationExp == 0 {
		config.VerificationExp = 5 * time.Minute
	}

	return &SMSCache{
		client: client,
		config: config,
	}
}

// SaveCode 保存验证码
func (c *SMSCache) SaveCode(ctx context.Context, phone, code string, expire time.Duration) error {
	if expire == 0 {
		expire = c.config.VerificationExp
	}
	key := c.getKey(fmt.Sprintf("verify:%s", phone))
	return c.client.Set(ctx, key, code, expire).Err()
}

// GetCode 获取验证码
func (c *SMSCache) GetCode(ctx context.Context, phone string) (string, error) {
	key := c.getKey(fmt.Sprintf("verify:%s", phone))
	return c.client.Get(ctx, key).Result()
}

// DeleteCode 删除验证码
func (c *SMSCache) DeleteCode(ctx context.Context, phone string) error {
	key := c.getKey(fmt.Sprintf("verify:%s", phone))
	return c.client.Del(ctx, key).Err()
}

// CheckLimit 检查发送频率限制
func (c *SMSCache) CheckLimit(ctx context.Context, phone string) (bool, error) {
	if !c.config.EnableLimit {
		return true, nil
	}

	key := c.getKey(fmt.Sprintf("limit:%s", phone))

	// 使用 Lua 脚本保证原子性
	script := `
		local count = redis.call("INCR", KEYS[1])
		if count == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		if count > tonumber(ARGV[2]) then
			return 0
		end
		return 1
	`

	windowSeconds := int64(c.config.LimitWindow.Seconds())
	result, err := c.client.Eval(ctx, script, []string{key}, windowSeconds, c.config.LimitCount).Int()
	if err != nil {
		return false, err
	}

	return result == 1, nil
}

// RecordAttempt 记录发送尝试
func (c *SMSCache) RecordAttempt(ctx context.Context, phone string) error {
	key := c.getKey(fmt.Sprintf("attempt:%s", phone))
	timestamp := time.Now().Unix()
	return c.client.LPush(ctx, key, strconv.FormatInt(timestamp, 10)).Err()
}

// SaveFailoverRecord 保存 Failover 记录
func (c *SMSCache) SaveFailoverRecord(ctx context.Context, phone, failedProvider, successProvider string) error {
	key := c.getKey(fmt.Sprintf("failover:%s", phone))
	record := fmt.Sprintf("%s->%s:%d", failedProvider, successProvider, time.Now().Unix())
	return c.client.RPush(ctx, key, record).Err()
}

// GetFailoverRecords 获取 Failover 记录
func (c *SMSCache) GetFailoverRecords(ctx context.Context, phone string, limit int) ([]string, error) {
	key := c.getKey(fmt.Sprintf("failover:%s", phone))
	if limit <= 0 {
		limit = 10
	}
	return c.client.LRange(ctx, key, 0, int64(limit-1)).Result()
}

// getKey 生成 Redis key
func (c *SMSCache) getKey(suffix string) string {
	return c.config.Prefix + suffix
}
