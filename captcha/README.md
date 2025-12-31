# Captcha 验证码模块

验证码模块提供图片验证码的生成、验证和管理功能，支持自定义配置和多种存储后端。

## 功能特性

- 生成图片验证码
- 验证码验证与管理
- Redis 存储支持
- 可配置的验证码参数
- Base64 格式图片输出

## 安装

```bash
go get -u github.com/linorwang/goaid
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/linorwang/goaid/captcha"
    "github.com/redis/go-redis/v9"
)

func main() {
    // 初始化 Redis 客户端
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 创建验证码存储
    captchaStore := captcha.NewRedisCaptchaStore(redisClient, "captcha:")

    // 配置验证码选项
    opts := captcha.CaptchaOption{
        ExpireTime: 5 * time.Minute,
        Length:     4,
        Width:      120,
        Height:     40,
    }

    // 创建验证码服务
    service := captcha.NewDefaultImageCaptchaService(captchaStore, opts)

    // 生成验证码
    ctx := context.Background()
    resp, err := service.GenerateImageCaptcha(ctx, 0, 0)
    if err != nil {
        fmt.Printf("生成验证码失败: %v\n", err)
        return
    }

    fmt.Printf("验证码ID: %s\n", resp.ID)
    fmt.Printf("Base64图片长度: %d\n", len(resp.ImageBase64))

    // 将 resp.ImageBase64 直接返回给前端，前端可以直接使用
    // 例如在HTML中: <img src="" + resp.ImageBase64 + "" alt="验证码">

    // 验证验证码
    isValid, err := service.VerifyCaptcha(ctx, resp.ID, "user_input")
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
        return
    }

    if isValid {
        fmt.Println("验证码验证成功")
    } else {
        fmt.Println("验证码验证失败")
    }
}
```

## 使用说明

### 1. 与已有 Redis 客户端集成

如果您的项目已经初始化了 Redis 客户端，可以直接使用：

```go
// 使用您已有的 Redis 客户端
captchaStore := captcha.NewRedisCaptchaStore(yourRedisClient, "myapp:captcha:")

// 创建验证码服务
captchaService := captcha.NewDefaultImageCaptchaService(captchaStore, captchaOpts)
```

### 2. 前端集成

前端可以直接使用返回的 `ImageBase64` 字段，无需额外解码：

```html
<!-- 直接使用 ImageBase64 作为图片的 src -->
<img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAHgAAABQCAYAAAC..." alt="验证码">
```

### 3. Web API 集成示例

在 Web 应用中，您可以创建如下 API 接口：

```go
// 生成验证码接口
func generateCaptchaHandler(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background()
    resp, err := captchaService.GenerateImageCaptcha(ctx, 0, 0)
    if err != nil {
        // 处理错误
        return
    }
    
    // 返回包含 ImageBase64 的 JSON 响应
    response := map[string]interface{}{
        "code": 200,
        "message": "success",
        "data": map[string]interface{}{
            "id": resp.ID,
            "image_base64": resp.ImageBase64, // 前端可直接使用的 base64 图片
            "expire_at": resp.ExpireAt.Unix(),
        },
    }
    
    json.NewEncoder(w).Encode(response)
}

// 验证验证码接口
func verifyCaptchaHandler(w http.ResponseWriter, r *http.Request) {
    // 解析请求数据
    var req struct {
        ID     string `json:"id"`
        Answer string `json:"answer"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    ctx := context.Background()
    isValid, err := captchaService.VerifyCaptcha(ctx, req.ID, req.Answer)
    if err != nil {
        // 处理错误
        return
    }
    
    response := map[string]interface{}{
        "code": 200,
        "message": "success",
        "data": isValid,
    }
    
    json.NewEncoder(w).Encode(response)
}
```

## API 文档

### CaptchaOption 配置选项

- `ExpireTime`: 验证码过期时间，默认 5 分钟
- `Length`: 验证码长度，默认 4 位
- `Width`: 图片宽度，默认 120 像素
- `Height`: 图片高度，默认 40 像素
- `Complexity`: 复杂度级别

### CaptchaResponse 响应结构

- `ID`: 验证码唯一标识符
- `Image`: 图片对象（用于内部处理）
- `ImageBase64`: Base64 编码的图片数据，前端可直接使用
- `ExpireAt`: 过期时间
- `Value`: 验证码值（仅用于测试，生产环境不应返回）

### 主要方法

- `GenerateImageCaptcha(ctx, width, height)`: 生成图片验证码
- `VerifyCaptcha(ctx, id, answer)`: 验证验证码
- `DeleteCaptcha(ctx, id)`: 删除验证码

## 注意事项

1. 生产环境中，不要返回 `Value` 字段给前端
2. 验证码验证成功后会自动删除，防止重复使用
3. 前端可以直接使用 `ImageBase64` 字段作为图片的 `src` 属性
4. 验证码 ID 需要与用户输入的验证码值一起提交到后端验证