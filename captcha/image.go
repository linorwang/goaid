package captcha

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
)

// DefaultImageCaptchaService 默认图片验证码服务
type DefaultImageCaptchaService struct {
	store   CaptchaStore
	options CaptchaOption
}

// NewDefaultImageCaptchaService 创建默认图片验证码服务
func NewDefaultImageCaptchaService(store CaptchaStore, options CaptchaOption) *DefaultImageCaptchaService {
	if options.ExpireTime == 0 {
		options.ExpireTime = 5 * time.Minute // 默认5分钟过期
	}
	if options.Length == 0 {
		options.Length = 4 // 默认4位验证码
	}
	if options.Width == 0 {
		options.Width = 120 // 默认宽度
	}
	if options.Height == 0 {
		options.Height = 40 // 默认高度
	}

	return &DefaultImageCaptchaService{
		store:   store,
		options: options,
	}
}

// GenerateImageCaptcha 生成图片验证码
func (s *DefaultImageCaptchaService) GenerateImageCaptcha(ctx context.Context, width, height int) (*CaptchaResponse, error) {
	if width == 0 {
		width = s.options.Width
	}
	if height == 0 {
		height = s.options.Height
	}

	// 创建数字验证码驱动
	driver := base64Captcha.NewDriverDigit(height, width, s.options.Length, 0.7, 80)

	// 创建验证码
	captcha := base64Captcha.NewCaptcha(driver, nil)

	// 生成验证码
	idKey, content, _ := captcha.Driver.GenerateIdQuestionAnswer()
	item, err := captcha.Driver.DrawCaptcha(content)
	if err != nil {
		return nil, err
	}

	// 将图片转换为base64字符串
	b64s := item.EncodeB64string()
		
	// 解码base64为图片字节
	// 移除base64字符串中的前缀部分，如 "data:image/png;base64," 
	if idx := strings.Index(b64s, ","); idx != -1 {
		b64s = b64s[idx+1:]
	}
		
	imgData, err := base64.StdEncoding.DecodeString(b64s)
	if err != nil {
		return nil, err
	}
		
	// 创建图片
	img, err := png.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}

	// 存储验证码到存储器
	err = s.store.Set(ctx, idKey, content, s.options.ExpireTime)
	if err != nil {
		return nil, err
	}

	// 将图片转换为base64字符串
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}
	imgBytes := buf.Bytes()
	base64Str := base64.StdEncoding.EncodeToString(imgBytes)

	expireAt := time.Now().Add(s.options.ExpireTime)

	response := &CaptchaResponse{
		ID:        idKey,
		Image:     img,
		ImageBase64: "data:image/png;base64," + base64Str,
		ExpireAt:  expireAt,
		// 注意：在生产环境中，不应返回Value字段，这里仅用于演示
		Value:     content,
	}

	return response, nil
}

// VerifyCaptcha 验证验证码
func (s *DefaultImageCaptchaService) VerifyCaptcha(ctx context.Context, id, answer string) (bool, error) {
	// 获取存储的验证码
	storedAnswer, err := s.store.Get(ctx, id)
	if err != nil {
		return false, err
	}

	// 验证答案是否正确
	isValid := strings.ToLower(storedAnswer) == strings.ToLower(answer)

	if isValid {
		// 验证成功后删除验证码（防止重复使用）
		s.store.Delete(ctx, id)
	}

	return isValid, nil
}

// DeleteCaptcha 删除验证码
func (s *DefaultImageCaptchaService) DeleteCaptcha(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}