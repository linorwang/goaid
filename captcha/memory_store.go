package captcha

import (
	"context"
	"sync"
	"time"
)

type memoryCaptchaItem struct {
	value    string
	expireAt time.Time
}

// MemoryCaptchaStore stores captchas in memory.
//
// It is useful for tests, local demos, and single-process services. For
// multi-instance production services, prefer RedisCaptchaStore.
type MemoryCaptchaStore struct {
	mu   sync.RWMutex
	data map[string]memoryCaptchaItem
}

// NewMemoryCaptchaStore creates an in-memory captcha store.
func NewMemoryCaptchaStore() *MemoryCaptchaStore {
	return &MemoryCaptchaStore{
		data: make(map[string]memoryCaptchaItem),
	}
}

// Set stores a captcha value.
func (m *MemoryCaptchaStore) Set(ctx context.Context, id string, value string, expire time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[id] = memoryCaptchaItem{
		value:    value,
		expireAt: time.Now().Add(expire),
	}
	return nil
}

// Get returns a captcha value.
func (m *MemoryCaptchaStore) Get(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	m.mu.RLock()
	item, ok := m.data[id]
	m.mu.RUnlock()
	if !ok {
		return "", ErrCaptchaNotFound
	}

	if time.Now().After(item.expireAt) {
		_ = m.Delete(ctx, id)
		return "", ErrCaptchaNotFound
	}

	return item.value, nil
}

// Delete removes a captcha value.
func (m *MemoryCaptchaStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, id)
	return nil
}
