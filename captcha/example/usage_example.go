package main

import (
	"context"
	"fmt"
	"time"

	"github.com/linorwang/goaid/captcha"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 假设您已经在外部项目中初始化了 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // 您的 Redis 地址
		Password: "",               // 您的 Redis 密码，如果有的话
		DB:       0,                // 使用的数据库
	})

	// 创建验证码存储，使用您已有的 Redis 客户端
	captchaStore := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")

	// 配置验证码选项
	captchaOpts := captcha.CaptchaOption{
		ExpireTime: 5 * time.Minute, // 5分钟过期
		Length:     4,               // 4位验证码
		Width:      120,             // 图片宽度
		Height:     40,              // 图片高度
		Complexity: 2,               // 复杂度
	}

	// 创建验证码服务
	captchaService := captcha.NewDefaultImageCaptchaService(captchaStore, captchaOpts)

	// 生成验证码
	ctx := context.Background()
	captchaResp, err := captchaService.GenerateImageCaptcha(ctx, 0, 0)
	if err != nil {
		fmt.Printf("生成验证码失败: %v\n", err)
		return
	}

	fmt.Printf("验证码ID: %s\n", captchaResp.ID)
	fmt.Printf("验证码值: %s\n", captchaResp.Value) // 仅用于调试，生产环境不应返回
	fmt.Printf("Base64图片长度: %d\n", len(captchaResp.ImageBase64))

	// 在您的应用中，直接将 captchaResp.ImageBase64 返回给前端，前端可以直接使用
	// 例如在HTML中: <img src="" + captchaResp.ImageBase64 + "" alt="验证码">
	// 用户输入验证码后，调用验证方法

	// 验证验证码（使用用户输入的值）
	userInput := "1234" // 这是用户实际输入的验证码
	isValid, err := captchaService.VerifyCaptcha(ctx, captchaResp.ID, userInput)
	if err != nil {
		fmt.Printf("验证验证码时出错: %v\n", err)
		return
	}

	if isValid {
		fmt.Println("验证码验证成功！可以继续发送短信")
		// 在这里您可以继续执行发送短信的逻辑
	} else {
		fmt.Println("验证码验证失败！")
	}
}

// 以下是一个在 Web 应用中的使用示例
func WebUsageExample() {
	// 初始化 Redis 客户端（在您的应用启动时完成）
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 创建验证码存储和服务（通常在应用初始化时创建一次）
	captchaStore := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")
	captchaOpts := captcha.CaptchaOption{
		ExpireTime: 5 * time.Minute,
		Length:     4,
		Width:      120,
		Height:     40,
	}
	captchaService := captcha.NewDefaultImageCaptchaService(captchaStore, captchaOpts)

	// 在处理生成验证码请求时
	generateCaptchaHandler := func() {
		ctx := context.Background()
		resp, err := captchaService.GenerateImageCaptcha(ctx, 0, 0)
		if err != nil {
			// 处理错误
			return
		}

		// 直接将 resp.ImageBase64 发送给前端，前端可以直接使用
		// 返回 resp.ID 给前端，用于后续验证
		fmt.Printf("验证码ID: %s\n", resp.ID)
		fmt.Printf("Base64图片长度: %d\n", len(resp.ImageBase64))
	}

	// 在处理验证验证码请求时
	verifyCaptchaHandler := func(captchaID, userInput string) bool {
		ctx := context.Background()
		isValid, err := captchaService.VerifyCaptcha(ctx, captchaID, userInput)
		if err != nil {
			fmt.Printf("验证出错: %v\n", err)
			return false
		}
		return isValid
	}

	// 调用示例
	generateCaptchaHandler()

	// 模拟用户验证
	result := verifyCaptchaHandler("some-captcha-id", "user-input")
	if result {
		fmt.Println("验证通过，可以执行后续操作")
	} else {
		fmt.Println("验证失败")
	}
}