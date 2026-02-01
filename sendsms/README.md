# 短信发送 SDK (sendsms)

一个功能强大、易于使用的 Go 语言短信发送 SDK，支持多服务商、自动容错、Failover 机制、验证码管理等功能。

## 特性

- **多服务商支持**：支持阿里云、腾讯云、百度云、华为云、网易云信、容联云、极光、创蓝253、Twilio 等主流短信服务商
- **自动 Failover**：主服务商失败时自动切换到备用服务商
- **智能重试**：支持固定延迟、指数退避、线性退避三种重试策略
- **验证码管理**：内置验证码生成、存储、验证功能
- **频率限制**：基于 Redis 的频率限制，防止恶意刷接口
- **批量发送**：支持批量发送和并发批量发送
- **健康检查**：实时监控服务商健康状态
- **易于扩展**：清晰的接口设计，方便添加新的服务商

## 安装

```bash
go get github.com/linorwang/goaid/sendsms
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "time"
    
    "github.com/redis/go-redis/v9"
    "github.com/linorwang/goaid/sendsms"
)

func main() {
    // 1. 创建 Redis 客户端（用于验证码存储和限流）
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // 2. 创建服务商实例
    providers := map[string]sendsms.SMSProvider{
        "aliyun":  NewAliyunProvider(/* 阿里云配置 */),
        "tencent": NewTencentProvider(/* 腾讯云配置 */),
    }
    
    // 3. 创建配置
    config := sendsms.DefaultConfig()
    config.PrimaryProvider = "aliyun"
    config.BackupProviders = []string{"tencent"}
    config.EnableFailover = true
    config.RetryTimes = 3
    
    // 4. 创建短信客户端
    client, err := sendsms.NewSMSClient(
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
    
    // 5. 发送短信
    resp, err := client.Send(ctx, &sendsms.SMSRequest{
        Phone:    "+8613800138000",
        Template: "SMS_123456",
        Params:   []string{"验证码", "1234", "5"},
        SignName: "我的签名",
        Type:     sendsms.SMSVerification,
    })
    
    if err != nil {
        fmt.Printf("发送失败: %v\n", err)
    } else {
        fmt.Printf("发送成功: %+v\n", resp)
    }
}
```

### 发送验证码

```go
// 发送验证码
resp, err := client.SendVerificationCode(ctx, &sendsms.VerificationCodeRequest{
    Phone:      "+8613800138000",
    Template:   "SMS_123456",
    SignName:   "我的签名",
    CodeLength: 6,
    ExpireTime: 5 * time.Minute,
})

// 验证验证码
result, err := client.VerifyCode(ctx, &sendsms.VerifyCodeRequest{
    Phone:     "+8613800138000",
    Code:      "123456",
    CleanOnce: true, // 验证后自动删除
})

if result.Valid {
    fmt.Println("验证码正确")
}
```

### 批量发送

```go
// 准备批量请求
reqs := []*sendsms.SMSRequest{
    {
        Phone:    "+8613800138000",
        Template: "SMS_123456",
        Params:   []string{"参数1", "参数2"},
        SignName: "我的签名",
    },
    // 更多请求...
}

// 批量发送（顺序）
result, err := client.SendBatch(ctx, reqs)

// 并发批量发送（控制并发数）
result, err := client.SendBatchConcurrent(ctx, reqs, 10)

fmt.Printf("成功: %d, 失败: %d\n", result.Success, result.Failed)
```

## 配置说明

### Config 结构

```go
type Config struct {
    // 服务商配置
    PrimaryProvider string   // 主服务商名称
    BackupProviders []string // 备用服务商列表
    
    // 默认设置
    DefaultSign     string // 默认签名
    DefaultTemplate string // 默认模板ID
    
    // 缓存配置
    CacheConfig *CacheConfig
    
    // 重试配置
    RetryStrategy   RetryStrategy // 重试策略
    RetryTimes      int           // 重试次数
    RetryDelay      time.Duration // 初始重试延迟
    MaxRetryDelay   time.Duration // 最大重试延迟
    RetryMultiplier float64       // 退避乘数
    
    // Failover 配置
    EnableFailover      bool           // 是否启用 Failover
    FailoverStrategy    FailoverStrategy // Failover 策略
    FailoverCooldown    time.Duration  // Failover 冷却时间
    HealthCheckInterval time.Duration  // 健康检查间隔
    
    // 请求配置
    Timeout         time.Duration // 请求超时
    BatchSize       int           // 批量发送大小
    ConcurrentLimit int           // 并发限制
}
```

### 重试策略

- `RetryFixedDelay`：固定延迟重试
- `RetryExponentialBackoff`：指数退避（推荐）
- `RetryLinearBackoff`：线性退避

### Failover 策略

- `FailoverSequential`：顺序切换（默认）
- `FailoverRandom`：随机选择
- `FailoverRoundRobin`：轮询

## 支持的服务商

| 服务商 | 状态 | 说明 |
|--------|------|------|
| 阿里云 | 待实现 | 阿里云短信服务 |
| 腾讯云 | 待实现 | 腾讯云短信服务 |
| 百度云 | 待实现 | 百度云 SMS |
| 华为云 | 待实现 | 华为云消息服务 |
| 网易云信 | 待实现 | 网易云信短信 |
| 容联云 | 待实现 | 容联云通讯 |
| 极光 | 待实现 | 极光推送短信 |
| 创蓝253 | 待实现 | 创蓝253短信 |
| Twilio | 待实现 | Twilio 国际短信 |

## 错误处理

```go
resp, err := client.Send(ctx, req)

if err != nil {
    if smsErr, ok := err.(*sendsms.SMSError); ok {
        // 获取错误类型
        switch smsErr.ErrorType {
        case sendsms.ErrorTypeNetwork:
            // 网络错误
        case sendsms.ErrorTypeProvider:
            // 服务商错误
        case sendsms.ErrorTypeRateLimit:
            // 限流错误
        // ...
        }
        
        // 判断是否可重试
        if smsErr.Retryable {
            // 可以重试
        }
    }
}
```

## 监控与健康检查

```go
// 获取所有服务商健康状态
healthStatus := client.GetHealthStatus()
for _, health := range healthStatus {
    fmt.Printf("服务商: %s\n", health.Name)
    fmt.Printf("  状态: %v\n", health.IsHealthy)
    fmt.Printf("  错误次数: %d\n", health.ErrorCount)
    fmt.Printf("  最后检查时间: %v\n", health.LastCheckTime)
}
```

## Mock Provider（用于测试）

```go
// 创建 Mock 服务商（用于测试）
mockProvider := sendsms.NewMockProvider(
    "mock",
    0.1,                   // 10% 失败率
    100*time.Millisecond,   // 模拟延迟
)

providers := map[string]sendsms.SMSProvider{
    "mock": mockProviderProvider,
}

// 使用 Mock 服务商创建客户端
client, _ := sendsms.NewSMSClient(
    "mock",
    []string{},
    providers,
    nil, // 可以不传 Redis
    config,
)
```

## 最佳实践

1. **使用 Failover**：配置主服务商和备用服务商，提高可用性
2. **合理设置重试**：建议使用指数退避策略，重试 3-5 次
3. **启用限流**：防止恶意刷接口，保护系统
4. **监控健康状态**：定期检查服务商健康状态，及时发现问题
5. **批量发送优化**：大量短信发送时使用并发批量发送，控制并发数
6. **验证码管理**：使用内置验证码功能，简化开发

## 注意事项

1. Redis 是必需的，用于验证码存储和频率限制
2. 各服务商的配置参数不同，请参考各服务商的文档
3. 建议在生产环境启用 Failover 和重试机制
4. 注意短信发送频率限制，避免被封禁
5. 定期检查服务商余额，避免余额不足导致发送失败

## License

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
