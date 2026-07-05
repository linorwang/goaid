package captcha

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCaptchaStore stores captcha answers in Redis.
type RedisCaptchaStore struct {
	client redis.Cmdable
	prefix string
}

// NewRedisCaptchaStore creates a Redis-backed captcha store.
//
// The client can be *redis.Client, *redis.ClusterClient, *redis.Ring, or any
// custom implementation of redis.Cmdable.
func NewRedisCaptchaStore(client redis.Cmdable, prefix string) *RedisCaptchaStore {
	if prefix == "" {
		prefix = "captcha:"
	}
	return &RedisCaptchaStore{
		client: client,
		prefix: prefix,
	}
}

// NewRedisService creates an image captcha service backed by Redis.
func NewRedisService(client redis.Cmdable, prefix string, options ...CaptchaOption) *DefaultImageCaptchaService {
	return New(NewRedisCaptchaStore(client, prefix), options...)
}

// Set stores a captcha answer.
func (r *RedisCaptchaStore) Set(ctx context.Context, id string, value string, expire time.Duration) error {
	return r.client.Set(ctx, r.key(id), value, expire).Err()
}

// Get returns a captcha answer.
func (r *RedisCaptchaStore) Get(ctx context.Context, id string) (string, error) {
	value, err := r.client.Get(ctx, r.key(id)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrCaptchaNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// Delete removes a captcha answer.
func (r *RedisCaptchaStore) Delete(ctx context.Context, id string) error {
	return r.client.Del(ctx, r.key(id)).Err()
}

func (r *RedisCaptchaStore) key(id string) string {
	return r.prefix + id
}
