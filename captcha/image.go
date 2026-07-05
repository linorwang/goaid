package captcha

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
)

const (
	defaultExpireTime = 5 * time.Minute
	defaultLength     = 4
	defaultWidth      = 120
	defaultHeight     = 40
	defaultComplexity = 80
)

// DefaultImageCaptchaService is the default image captcha implementation.
type DefaultImageCaptchaService struct {
	store   CaptchaStore
	options CaptchaOption
}

// New creates an image captcha service with defaults.
//
// Passing nil as store creates an in-memory store, which is convenient for
// tests and single-process demos. Use RedisCaptchaStore for production.
func New(store CaptchaStore, options ...CaptchaOption) *DefaultImageCaptchaService {
	option := CaptchaOption{}
	if len(options) > 0 {
		option = options[0]
	}
	return NewDefaultImageCaptchaService(store, option)
}

// NewDefaultImageCaptchaService creates the default image captcha service.
func NewDefaultImageCaptchaService(store CaptchaStore, options CaptchaOption) *DefaultImageCaptchaService {
	if store == nil {
		store = NewMemoryCaptchaStore()
	}

	return &DefaultImageCaptchaService{
		store:   store,
		options: withDefaults(options),
	}
}

func withDefaults(options CaptchaOption) CaptchaOption {
	if options.ExpireTime <= 0 {
		options.ExpireTime = defaultExpireTime
	}
	if options.Length <= 0 {
		options.Length = defaultLength
	}
	if options.Width <= 0 {
		options.Width = defaultWidth
	}
	if options.Height <= 0 {
		options.Height = defaultHeight
	}
	if options.Complexity <= 0 {
		options.Complexity = defaultComplexity
	}
	return options
}

// Generate creates a captcha with the configured default size.
func (s *DefaultImageCaptchaService) Generate(ctx context.Context) (*CaptchaResponse, error) {
	return s.GenerateImageCaptcha(ctx, 0, 0)
}

// GenerateImageCaptcha creates an image captcha. Width and height fall back to
// service defaults when they are less than or equal to zero.
func (s *DefaultImageCaptchaService) GenerateImageCaptcha(ctx context.Context, width, height int) (*CaptchaResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if width <= 0 {
		width = s.options.Width
	}
	if height <= 0 {
		height = s.options.Height
	}

	driver := base64Captcha.NewDriverDigit(height, width, s.options.Length, 0.7, s.options.Complexity)
	captcha := base64Captcha.NewCaptcha(driver, nil)

	id, answer, _ := captcha.Driver.GenerateIdQuestionAnswer()
	item, err := captcha.Driver.DrawCaptcha(answer)
	if err != nil {
		return nil, err
	}

	imageBase64 := item.EncodeB64string()
	imageData := imageBase64
	if idx := strings.Index(imageData, ","); idx != -1 {
		imageData = imageData[idx+1:]
	}

	imgBytes, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}

	if err := s.store.Set(ctx, id, answer, s.options.ExpireTime); err != nil {
		return nil, err
	}

	response := &CaptchaResponse{
		ID:          id,
		Image:       img,
		ImageBase64: imageBase64,
		ExpireAt:    time.Now().Add(s.options.ExpireTime),
	}
	if s.options.IncludeValue {
		response.Value = answer
	}

	return response, nil
}

// Verify verifies a captcha with the configured store.
func (s *DefaultImageCaptchaService) Verify(ctx context.Context, id, answer string) (bool, error) {
	return s.VerifyCaptcha(ctx, id, answer)
}

// VerifyCaptcha verifies a captcha. Missing, expired, or empty values return
// false with nil error so HTTP handlers can use the boolean directly.
func (s *DefaultImageCaptchaService) VerifyCaptcha(ctx context.Context, id, answer string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	id = strings.TrimSpace(id)
	answer = strings.TrimSpace(answer)
	if id == "" || answer == "" {
		return false, nil
	}

	storedAnswer, err := s.store.Get(ctx, id)
	if errors.Is(err, ErrCaptchaNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	isValid := strings.EqualFold(strings.TrimSpace(storedAnswer), answer)
	if isValid {
		if err := s.store.Delete(ctx, id); err != nil {
			return false, err
		}
	}

	return isValid, nil
}

// DeleteCaptcha deletes a captcha from the store.
func (s *DefaultImageCaptchaService) DeleteCaptcha(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
