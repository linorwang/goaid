package captcha

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockCaptchaStore 模拟验证码存储用于测试
type MockCaptchaStore struct {
	data map[string]string
	ttl  map[string]time.Time
}

func NewMockCaptchaStore() *MockCaptchaStore {
	return &MockCaptchaStore{
		data: make(map[string]string),
		ttl:  make(map[string]time.Time),
	}
}

func (m *MockCaptchaStore) Set(ctx context.Context, id string, value string, expire time.Duration) error {
	m.data[id] = value
	m.ttl[id] = time.Now().Add(expire)
	return nil
}

func (m *MockCaptchaStore) Get(ctx context.Context, id string) (string, error) {
	if expiry, exists := m.ttl[id]; exists && time.Now().After(expiry) {
		delete(m.data, id)
		delete(m.ttl, id)
		return "", fmt.Errorf("captcha not found")
	}
	
	if value, exists := m.data[id]; exists {
		return value, nil
	}
	return "", fmt.Errorf("captcha not found")
}

func (m *MockCaptchaStore) Delete(ctx context.Context, id string) error {
	delete(m.data, id)
	delete(m.ttl, id)
	return nil
}

func TestImageCaptchaService(t *testing.T) {
	store := NewMockCaptchaStore()
	
	opts := CaptchaOption{
		ExpireTime: 5 * time.Minute,
		Length:     4,
		Width:      120,
		Height:     40,
	}
	
	service := NewDefaultImageCaptchaService(store, opts)
	
	// 测试生成验证码
	ctx := context.Background()
	resp, err := service.GenerateImageCaptcha(ctx, 120, 40)
	if err != nil {
		t.Fatalf("GenerateImageCaptcha failed: %v", err)
	}
	
	if resp.ID == "" {
		t.Fatal("Expected captcha ID, got empty")
	}
	
	if resp.Value == "" {
		t.Fatal("Expected captcha value, got empty")
	}
		
	if resp.ImageBase64 == "" {
		t.Fatal("Expected captcha image base64, got empty")
	}
		
	// 测试验证验证码
	isValid, err := service.VerifyCaptcha(ctx, resp.ID, resp.Value)
	if err != nil {
		t.Fatalf("VerifyCaptcha failed: %v", err)
	}
	
	if !isValid {
		t.Error("Expected captcha to be valid")
	}
	
	// 由于验证码已验证成功并被删除，现在应该找不到该验证码
	isValid, err = service.VerifyCaptcha(ctx, resp.ID, resp.Value)
	if err == nil {
		t.Error("Expected error when verifying same captcha again")
	}
	
	// 生成另一个验证码用于测试错误答案
	resp2, err := service.GenerateImageCaptcha(ctx, 120, 40)
	if err != nil {
		t.Fatalf("GenerateImageCaptcha failed: %v", err)
	}
	
	// 测试错误的验证码
	isValid, err = service.VerifyCaptcha(ctx, resp2.ID, "wrong")
	if err != nil {
		t.Errorf("VerifyCaptcha failed with wrong answer: %v", err)
	}
	
	if isValid {
		t.Error("Expected captcha to be invalid with wrong answer")
	}
}

func TestCaptchaExpiration(t *testing.T) {
	store := NewMockCaptchaStore()
	
	opts := CaptchaOption{
		ExpireTime: 100 * time.Millisecond, // 短暂过期时间用于测试
		Length:     4,
		Width:      120,
		Height:     40,
	}
	
	service := NewDefaultImageCaptchaService(store, opts)
	
	ctx := context.Background()
	resp, err := service.GenerateImageCaptcha(ctx, 120, 40)
	if err != nil {
		t.Errorf("GenerateImageCaptcha failed: %v", err)
	}
	
	// 等待过期
	time.Sleep(200 * time.Millisecond)
	
	// 验证过期的验证码应该失败
	_, err = service.VerifyCaptcha(ctx, resp.ID, resp.Value)
	if err == nil {
		t.Error("Expected error when verifying expired captcha")
	}
}

