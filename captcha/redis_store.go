package captcha

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCaptchaStore 基于Redis的验证码存储
type RedisCaptchaStore struct {
	client redis.Cmdable // 使用 redis.Cmdable 接口，支持单机、集群、哨兵等模式
	prefix string        // 键前缀
}

// NewRedisCaptchaStore 创建Redis验证码存储
// 参数:
//   - client: Redis 客户端实例，支持 redis.Cmdable 接口的任何实现
//   - prefix: Redis 键前缀（可选，默认为 "captcha:"）
//
// 支持的 Redis 客户端类型:
//   - *redis.Client (单机模式)
//   - *redis.ClusterClient (集群模式)
//   - *redis.Ring (哨兵模式)
//   - 任何实现了 redis.Cmdable 接口的自定义客户端
func NewRedisCaptchaStore(client redis.Cmdable, prefix string) *RedisCaptchaStore {
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
