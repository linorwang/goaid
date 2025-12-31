package captcha

import (
	"context"
	"image"
	"time"
)

// CaptchaStore 验证码存储接口
type CaptchaStore interface {
	Set(ctx context.Context, id string, value string, expire time.Duration) error
	Get(ctx context.Context, id string) (string, error)
	Delete(ctx context.Context, id string) error
}

// ImageCaptchaService 图片验证码服务接口
type ImageCaptchaService interface {
	GenerateImageCaptcha(ctx context.Context, width, height int) (*CaptchaResponse, error)
	VerifyCaptcha(ctx context.Context, id, answer string) (bool, error)
	DeleteCaptcha(ctx context.Context, id string) error
}

// CaptchaOption 验证码配置选项
type CaptchaOption struct {
	ExpireTime time.Duration // 过期时间
	Length     int          // 验证码长度
	Width      int          // 图片宽度
	Height     int          // 图片高度
	Complexity int          // 复杂度级别
}

// CaptchaResponse 验证码响应结构
type CaptchaResponse struct {
	ID       string      `json:"id"`
	Image    image.Image `json:"-"`      // 图片数据
	ImageURL string      `json:"image_url,omitempty"` // 图片URL（可选）
	ImageBase64 string   `json:"image_base64"`        // base64格式的图片数据
	Value    string      `json:"value,omitempty"`     // 验证码值（仅用于测试，生产环境不应返回）
	ExpireAt time.Time   `json:"expire_at"`
}