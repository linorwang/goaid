package captcha

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCaptchaStore 基于Redis的验证码存储
type RedisCaptchaStore struct {
	client *redis.Client
	prefix string // 键前缀
}

// NewRedisCaptchaStore 创建Redis验证码存储
func NewRedisCaptchaStore(client *redis.Client, prefix string) *RedisCaptchaStore {
	if prefix == "" {
		prefix = "captcha:"
	}
	return &RedisCaptchaStore{
		client: client,
		prefix: prefix,
	}
}

// Set 存储验证码
func (r *RedisCaptchaStore) Set(ctx context.Context, id string, value string, expire time.Duration) error {
	key := r.prefix + id
	return r.client.Set(ctx, key, value, expire).Err()
}

// Get 获取验证码
func (r *RedisCaptchaStore) Get(ctx context.Context, id string) (string, error) {
	key := r.prefix + id
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("captcha not found")
		}
		return "", err
	}
	return value, nil
}

// Delete 删除验证码
func (r *RedisCaptchaStore) Delete(ctx context.Context, id string) error {
	key := r.prefix + id
	return r.client.Del(ctx, key).Err()
}