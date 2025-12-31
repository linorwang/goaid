package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/linorwang/goaid/captcha"
	"github.com/redis/go-redis/v9"
)

// CaptchaHandler 验证码处理器
type CaptchaHandler struct {
	service captcha.ImageCaptchaService
}

// CaptchaGenerateRequest 生成验证码请求
type CaptchaGenerateRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// CaptchaGenerateResponse 生成验证码响应
type CaptchaGenerateResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    *captchaResponse `json:"data,omitempty"`
}

// captchaResponse 验证码响应数据
type captchaResponse struct {
	ID        string `json:"id"`
	ImageBase64 string `json:"image_base64"`
	ExpireAt  int64  `json:"expire_at"`
}

// CaptchaVerifyRequest 验证验证码请求
type CaptchaVerifyRequest struct {
	ID     string `json:"id"`
	Answer string `json:"answer"`
}

// CaptchaVerifyResponse 验证验证码响应
type CaptchaVerifyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    bool   `json:"data,omitempty"`
}

// NewCaptchaHandler 创建验证码处理器
func NewCaptchaHandler(service captcha.ImageCaptchaService) *CaptchaHandler {
	return &CaptchaHandler{
		service: service,
	}
}

// GenerateCaptcha 生成验证码
func (h *CaptchaHandler) GenerateCaptcha(w http.ResponseWriter, r *http.Request) {
	var req CaptchaGenerateRequest
	
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	
	// 解析请求参数
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// 生成验证码
	ctx := context.Background()
	resp, err := h.service.GenerateImageCaptcha(ctx, req.Width, req.Height)
	if err != nil {
		log.Printf("生成验证码失败: %v", err)
		json.NewEncoder(w).Encode(CaptchaGenerateResponse{
			Code:    500,
			Message: "生成验证码失败",
		})
		return
	}
	
	// 返回响应
	response := CaptchaGenerateResponse{
		Code:    200,
		Message: "success",
		Data: &captchaResponse{
			ID:        resp.ID,
			ImageBase64: resp.ImageBase64,
			ExpireAt:  resp.ExpireAt.Unix(),
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// VerifyCaptcha 验证验证码
func (h *CaptchaHandler) VerifyCaptcha(w http.ResponseWriter, r *http.Request) {
	var req CaptchaVerifyRequest
	
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	
	// 解析请求参数
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// 验证验证码
	ctx := context.Background()
	isValid, err := h.service.VerifyCaptcha(ctx, req.ID, req.Answer)
	if err != nil {
		log.Printf("验证验证码失败: %v", err)
		json.NewEncoder(w).Encode(CaptchaVerifyResponse{
			Code:    500,
			Message: "验证验证码失败",
		})
		return
	}
	
	// 返回响应
	response := CaptchaVerifyResponse{
		Code:    200,
		Message: "success",
		Data:    isValid,
	}
	
	json.NewEncoder(w).Encode(response)
}

func main() {
	// 初始化 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 创建验证码存储
	captchaStore := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")

	// 配置验证码选项
	captchaOpts := captcha.CaptchaOption{
		ExpireTime: 5 * time.Minute,
		Length:     4,
		Width:      120,
		Height:     40,
	}

	// 创建验证码服务
	captchaService := captcha.NewDefaultImageCaptchaService(captchaStore, captchaOpts)

	// 创建验证码处理器
	handler := NewCaptchaHandler(captchaService)

	// 注册路由
	http.HandleFunc("/captcha/generate", handler.GenerateCaptcha)
	http.HandleFunc("/captcha/verify", handler.VerifyCaptcha)

	// 启动服务器
	fmt.Println("验证码服务启动在 :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}