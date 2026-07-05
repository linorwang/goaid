package captcha

import (
	"context"
	"errors"
	"image"
	"time"
)

var (
	// ErrCaptchaNotFound means the captcha does not exist or has expired.
	ErrCaptchaNotFound = errors.New("captcha not found")
)

// CaptchaStore is the storage interface used by captcha services.
type CaptchaStore interface {
	Set(ctx context.Context, id string, value string, expire time.Duration) error
	Get(ctx context.Context, id string) (string, error)
	Delete(ctx context.Context, id string) error
}

// ImageCaptchaService describes the stable image captcha API.
type ImageCaptchaService interface {
	GenerateImageCaptcha(ctx context.Context, width, height int) (*CaptchaResponse, error)
	VerifyCaptcha(ctx context.Context, id, answer string) (bool, error)
	DeleteCaptcha(ctx context.Context, id string) error
}

// CaptchaOption configures image captcha generation.
type CaptchaOption struct {
	ExpireTime   time.Duration // Expiration time. Default: 5 minutes.
	Length       int           // Captcha length. Default: 4.
	Width        int           // Image width. Default: 120.
	Height       int           // Image height. Default: 40.
	Complexity   int           // Noise complexity. Default: 80.
	IncludeValue bool          // Include the answer in CaptchaResponse.Value. Use only in tests.
}

// CaptchaResponse is returned after generating an image captcha.
type CaptchaResponse struct {
	ID          string      `json:"id"`
	Image       image.Image `json:"-"`
	ImageURL    string      `json:"image_url,omitempty"`
	ImageBase64 string      `json:"image_base64"`
	Value       string      `json:"value,omitempty"`
	ExpireAt    time.Time   `json:"expire_at"`
}
