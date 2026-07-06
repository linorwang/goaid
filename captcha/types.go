package captcha

import (
	"context"
	"image"
	"time"
)

// CaptchaType defines the image captcha generator type.
type CaptchaType string

const (
	// CaptchaTypeDigit generates numeric captcha codes. This is the default.
	CaptchaTypeDigit CaptchaType = "digit"
	// CaptchaTypeString generates captcha codes from CharacterSource.
	CaptchaTypeString CaptchaType = "string"
)

const (
	// CaptchaSourceDigits contains numeric characters for string captchas.
	CaptchaSourceDigits = "0123456789"
	// CaptchaSourceLetters contains upper and lower case English letters.
	CaptchaSourceLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// CaptchaSourceAlphaNumeric contains digits and English letters.
	CaptchaSourceAlphaNumeric = CaptchaSourceDigits + CaptchaSourceLetters
)

// CaptchaStore stores captcha values.
type CaptchaStore interface {
	Set(ctx context.Context, id string, value string, expire time.Duration) error
	Get(ctx context.Context, id string) (string, error)
	Delete(ctx context.Context, id string) error
}

// ImageCaptchaService generates and verifies image captchas.
type ImageCaptchaService interface {
	GenerateImageCaptcha(ctx context.Context, width, height int) (*CaptchaResponse, error)
	VerifyCaptcha(ctx context.Context, id, answer string) (bool, error)
	DeleteCaptcha(ctx context.Context, id string) error
}

// CaptchaOption configures image captcha generation.
type CaptchaOption struct {
	ExpireTime      time.Duration // expiration duration
	Length          int           // captcha code length
	Width           int           // image width in pixels
	Height          int           // image height in pixels
	Complexity      int           // noise count for string captchas
	Type            CaptchaType   // captcha type, default CaptchaTypeDigit
	CharacterSource string        // character source for string captchas
}

// CaptchaResponse is the generated captcha payload.
type CaptchaResponse struct {
	ID          string      `json:"id"`
	Image       image.Image `json:"-"`
	ImageURL    string      `json:"image_url,omitempty"`
	ImageBase64 string      `json:"image_base64"`
	Value       string      `json:"value,omitempty"` // for tests only; do not return to clients in production
	ExpireAt    time.Time   `json:"expire_at"`
}
