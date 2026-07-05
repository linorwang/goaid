# captcha

图片验证码工具包，支持生成 PNG base64 验证码、校验答案、自动过期和校验成功后自动删除。

## 安装

```bash
go get github.com/linorwang/goaid
```

## 最简单用法

本地测试或单进程服务可以直接使用内存存储：

```go
package main

import (
	"context"
	"fmt"

	"github.com/linorwang/goaid/captcha"
)

func main() {
	ctx := context.Background()
	service := captcha.New(nil)

	resp, err := service.Generate(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.ID)
	fmt.Println(resp.ImageBase64) // 可直接作为 <img src="..."> 使用

	ok, err := service.Verify(ctx, resp.ID, "用户输入的验证码")
	if err != nil {
		panic(err)
	}
	fmt.Println(ok)
}
```

## Redis 用法

多实例或生产环境建议使用 Redis：

```go
redisClient := redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

service := captcha.NewRedisService(redisClient, "myapp:captcha:", captcha.CaptchaOption{
	ExpireTime: 5 * time.Minute,
	Length:     4,
	Width:      120,
	Height:     40,
})
```

也可以分开创建 store：

```go
store := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")
service := captcha.New(store)
```

## HTTP 示例

生成接口只返回 `id` 和 `image_base64`，不要把答案返回给前端：

```go
func generateCaptcha(w http.ResponseWriter, r *http.Request) {
	resp, err := service.Generate(r.Context())
	if err != nil {
		http.Error(w, "generate captcha failed", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":           resp.ID,
		"image_base64": resp.ImageBase64,
		"expire_at":    resp.ExpireAt,
	})
}
```

校验接口直接根据布尔值判断即可。验证码不存在、已过期或答案错误都会返回 `false, nil`：

```go
func verifyCaptcha(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ok, err := service.Verify(r.Context(), req.ID, req.Answer)
	if err != nil {
		http.Error(w, "verify captcha failed", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": ok,
	})
}
```

## API

- `captcha.New(store, options...)`：创建验证码服务。`store` 为 `nil` 时使用内存存储。
- `captcha.NewRedisService(client, prefix, options...)`：创建 Redis 版验证码服务。
- `service.Generate(ctx)`：按默认尺寸生成验证码。
- `service.GenerateImageCaptcha(ctx, width, height)`：按指定尺寸生成验证码。
- `service.Verify(ctx, id, answer)`：校验验证码。
- `service.VerifyCaptcha(ctx, id, answer)`：兼容旧方法名。
- `service.DeleteCaptcha(ctx, id)`：手动删除验证码。

## 配置项

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `ExpireTime` | `5 * time.Minute` | 验证码有效期 |
| `Length` | `4` | 验证码长度 |
| `Width` | `120` | 图片宽度 |
| `Height` | `40` | 图片高度 |
| `Complexity` | `80` | 干扰复杂度 |
| `IncludeValue` | `false` | 是否在响应中带上答案，仅建议测试使用 |

## 注意事项

- 生产环境不要开启 `IncludeValue`，也不要把 `resp.Value` 返回给前端。
- 校验成功后验证码会自动删除，不能重复使用。
- 内存存储只适合测试、演示或单进程服务；多实例部署请使用 Redis。
- 前端可以直接使用 `resp.ImageBase64`：`<img src="data.image_base64">`。
