package sendsms

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Example 基本使用示例
func Example() {
	// 1. 创建 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// 2. 创建服务商实例
	providers := map[string]SMSProvider{
		"aliyun":  NewMockProvider("aliyun", 0.1, 100*time.Millisecond),   // 10% 失败率
		"tencent": NewMockProvider("tencent", 0.05, 150*time.Millisecond), // 5% 失败率
	}

	// 3. 创建配置
	config := DefaultConfig()
	config.PrimaryProvider = "aliyun"
	config.BackupProviders = []string{"tencent"}
	config.EnableFailover = true
	config.RetryTimes = 3
	config.CacheConfig.EnableLimit = true
	config.CacheConfig.LimitCount = 5
	config.CacheConfig.LimitWindow = time.Hour

	// 4. 创建短信客户端
	client, err := NewSMSClient(
		"aliyun",
		[]string{"tencent"},
		providers,
		redisClient,
		config,
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// 示例1: 发送单条短信
	resp, err := client.Send(ctx, &SMSRequest{
		Phone:    "+8613800138000",
		Template: "SMS_123456",
		Params:   []string{"验证码", "1234", "5"},
		SignName: "我的签名",
		Type:     SMSVerification,
	})
	if err != nil {
		fmt.Printf("发送失败: %v\n", err)
	} else {
		fmt.Printf("发送成功: %+v\n", resp)
	}

	// 示例2: 发送验证码
	verifyResp, err := client.SendVerificationCode(ctx, &VerificationCodeRequest{
		Phone:      "+8613800138000",
		Template:   "SMS_123456",
		SignName:   "我的签名",
		CodeLength: 6,
		ExpireTime: 5 * time.Minute,
	})
	if err != nil {
		fmt.Printf("发送验证码失败: %v\n", err)
	} else {
		fmt.Printf("发送验证码成功: %+v\n", verifyResp)
	}

	// 示例3: 验证验证码
	verifyResult, err := client.VerifyCode(ctx, &VerifyCodeRequest{
		Phone:     "+8613800kt8000",
		Code:      "123456",
		CleanOnce: true,
	})
	if err != nil {
		fmt.Printf("验证失败: %v\n", err)
	} else {
		fmt.Printf("验证结果: %+v\n", verifyResult)
	}

	// 示例4: 批量发送
	batchReqs := []*SMSRequest{
		{
			Phone:    "+8613800138000",
			Template: "SMS_123456",
			Params:   []string{"验证码", "1234", "5"},
			SignName: "我的签名",
		},
		{
			Phone:    "+8613900139000",
			Template: "SMS_123456",
			Params:   []string{"验证码", "5678", "5"},
			SignName: "我的签名",
		},
	}
	batchResult, err := client.SendBatch(ctx, batchReqs)
	if err != nil {
		fmt.Printf("批量发送失败: %v\n", err)
	} else {
		fmt.Printf("批量发送结果: 成功=%d, 失败=%d\n", batchResult.Success, batchResult.Failed)
	}

	// 示例5: 获取服务商健康状态
	healthStatus := client.GetHealthStatus()
	for _, health := range healthStatus {
		fmt.Printf("服务商 %s: 健康=%v, 错误次数=%d\n", health.Name, health.IsHealthy, health.ErrorCount)
	}

	// 示例6: 使用简化接口发送
	simpleResp, err := client.SendWithTemplate(
		ctx,
		"+8613800138000",
		"SMS_123456",
		"我的签名",
		[]string{"参数1", "参数2"},
	)
	if err != nil {
		fmt.Printf("发送失败: %v\n", err)
	} else {
		fmt.Printf("发送成功: %+v\n", simpleResp)
	}
}

// ExampleWithoutFailover 不使用 Failover 的示例
func ExampleWithoutFailover() {
	// 创建配置，禁用 Failover
	config := DefaultConfig()
	config.EnableFailover = false
	config.RetryTimes = 3

	// 创建服务商
	providers := map[string]SMSProvider{
		"aliyun": NewMockProvider("aliyun", 0.0, 0),
		// 其他服务商不会被使用
	}

	// 创建客户端
	client, err := NewSMSClient(
		"aliyun",
		[]string{}, // 即使指定了备用，也不会被使用
		providers,
		nil, // 可以不传 Redis
		config,
	)
	if err != nil {
		panic(err)
	}

	// 使用客户端...
	_ = client
}

// ExampleConcurrentBatch 并发批量发送示例
func ExampleConcurrentBatch() {
	// 初始化客户端...
	_ = time.Second
	var client *SMSClient

	// 准备批量请求
	reqs := make([]*SMSRequest, 1000)
	for i := 0; i < 1000; i++ {
		reqs[i] = &SMSRequest{
			Phone:    fmt.Sprintf("+861380013%04d", i),
			Template: "SMS_123456",
			Params:   []string{"验证码", "1234", "5"},
			SignName: "我的签名",
		}
	}

	ctx := context.Background()

	// 并发批量发送，控制并发数为 10
	result, err := client.SendBatchConcurrent(ctx, reqs, 10)
	if err != nil {
		fmt.Printf("并发批量发送失败: %v\n", err)
		return
	}

	fmt.Printf("并发批量发送结果: 总数=%d, 成功=%d, 失败=%d\n",
		result.Total, result.Success, result.Failed)

	// 如果有失败的，可以重试
	if result.Failed > 0 {
		retryResult, err := client.RetryFailed(ctx, result.FailedReqs)
		if err == nil {
			fmt.Printf("重试结果: 成功=%d, 失败=%d\n", retryResult.Success, retryResult.Failed)
		}
	}
}
