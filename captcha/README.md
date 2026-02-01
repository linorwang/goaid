# Captcha 验证码模块

一个简单的图片验证码工具包，帮你快速实现验证码功能。

## 📦 安装

```bash
go get -u github.com/linorwang/goaid
```

## 🚀 快速开始（3 步搞定）

### 第一步：导入包

```go
import (
    "github.com/linorwang/goaid/captcha"
    "github.com/redis/go-redis/v9"
)
```

### 第二步：初始化验证码服务

```go
// 使用你已有的 Redis 客户端
redisClient := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})

// 创建验证码服务
captchaStore := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")
captchaService := captcha.NewDefaultImageCaptchaService(captchaStore, captcha.CaptchaOption{
    ExpireTime: 5 * time.Minute, // 5 分钟过期
    Length:     4,               // 4 位数字
    Width:      120,             // 图片宽度
    Height:     40,              // 图片高度
})
```

### 第三步：使用验证码

**生成验证码：**
```go
ctx := context.Background()
resp, err := captchaService.GenerateImageCaptcha(ctx, 0, 0)
if err != nil {
    // 处理错误
}

// 返回给前端的数据
fmt.Println("验证码ID:", resp.ID)              // 保存到前端，用于验证
fmt.Println("图片数据:", resp.ImageBase64)      // 直接给前端显示图片
```

**验证验证码：**
```go
isValid, err := captchaService.VerifyCaptcha(ctx, resp.ID, "用户输入的验证码")
if err != nil {
    // 处理错误
}

if isValid {
    fmt.Println("验证成功！")
} else {
    fmt.Println("验证失败！")
}
```

## 💡 与你的项目集成

如果你的项目中已经有 `ioc.InitRedis()` 方法，这样用：

```go
// 获取你已有的 Redis 客户端
redisClient := ioc.InitRedis()  // 返回 redis.Cmdable 类型

// 直接使用，不需要类型转换
captchaStore := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")
```

**兼容说明：**
- ✅ 完全兼容 `redis.Cmdable` 接口
- ✅ 支持单机、集群、哨兵等所有 Redis 模式
- ✅ 无需任何类型转换

## 🔄 验证码使用流程

### 前端流程（登录页面）

```
用户访问登录页面
    ↓
页面加载（window.onload）
    ↓
自动请求生成验证码接口
    ↓
后端返回验证码ID和图片
    ↓
前端保存ID，显示图片
    ↓
用户输入验证码
    ↓
点击登录按钮
    ↓
先验证验证码是否正确
    ↓
验证成功？
    ├─ 是 → 提交登录请求
    │       ↓
    │   登录成功？
    │       ├─ 是 → 跳转首页
    │       └─ 否 → 刷新验证码，提示错误
    │
    └─ 否 → 刷新验证码，提示错误
```

### 关键要点

⚠️ **必须在页面加载时就请求验证码**
- 在 `window.onload` 中调用生成接口
- 确保用户看到页面时验证码已准备好
- 不要等用户点击才生成

⚠️ **验证码ID必须保存到全局变量**
- 后端返回的ID用于后续验证
- 每次生成验证码都要更新ID
- 验证时使用正确的ID

⚠️ **验证失败后必须刷新验证码**
- 防止暴力破解
- 提高安全性
- 给用户重新尝试的机会

⚠️ **验证成功后验证码会被自动删除**
- 同一个验证码不能重复使用
- 需要重新生成新验证码

### 完整的登录页面示例

查看 `captcha/example/login_with_captcha.html` 获取完整的登录页面示例，包含：
- 页面加载时自动生成验证码
- 点击图片刷新验证码
- 验证失败自动刷新
- 完整的错误处理
- 详细的代码注释

## 🌐 Web 应用示例

### 后端代码

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/linorwang/goaid/captcha"
    "github.com/redis/go-redis/v9"
)

// 全局验证码服务
var captchaService captcha.ImageCaptchaService

func main() {
    // 初始化 Redis（使用你已有的客户端）
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 初始化验证码服务
    captchaStore := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")
    captchaService = captcha.NewDefaultImageCaptchaService(captchaStore, captcha.CaptchaOption{
        ExpireTime: 5 * time.Minute,
        Length:     4,
        Width:      120,
        Height:     40,
    })

    // 注册接口
    http.HandleFunc("/api/captcha/generate", generateHandler)
    http.HandleFunc("/api/captcha/verify", verifyHandler)

    http.ListenAndServe(":8080", nil)
}

// 生成验证码接口
func generateHandler(w http.ResponseWriter, r *http.Request) {
    resp, err := captchaService.GenerateImageCaptcha(context.Background(), 0, 0)
    if err != nil {
        http.Error(w, "生成失败", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "code": 200,
        "data": map[string]string{
            "id":           resp.ID,
            "image_base64": resp.ImageBase64,
        },
    })
}

// 验证验证码接口
func verifyHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        ID     string `json:"id"`
        Answer string `json:"answer"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    isValid, _ := captchaService.VerifyCaptcha(context.Background(), req.ID, req.Answer)

    json.NewEncoder(w).Encode(map[string]interface{}{
        "code": 200,
        "data": isValid,
    })
}
```

### 前端代码（HTML + JavaScript）

```html
<!DOCTYPE html>
<html>
<head>
    <title>验证码示例</title>
</head>
<body>
    <div>
        <!-- 验证码图片 -->
        <img id="captcha-img" src="" />
        
        <!-- 刷新按钮 -->
        <button onclick="refreshCaptcha()">刷新</button>
        
        <!-- 输入框 -->
        <input type="text" id="captcha-input" placeholder="输入验证码" />
        <button onclick="verifyCaptcha()">验证</button>
        
        <!-- 提示信息 -->
        <p id="message"></p>
    </div>

    <script>
        let captchaId = '';

        // 生成验证码
        async function refreshCaptcha() {
            const res = await fetch('/api/captcha/generate', {
                method: 'POST'
            });
            const data = await res.json();
            
            captchaId = data.data.id;
            document.getElementById('captcha-img').src = data.data.image_base64;
        }

        // 验证验证码
        async function verifyCaptcha() {
            const input = document.getElementById('captcha-input').value;
            const res = await fetch('/api/captcha/verify', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    id: captchaId,
                    answer: input
                })
            });
            const data = await res.json();
            
            if (data.data) {
                document.getElementById('message').textContent = '✅ 验证成功';
            } else {
                document.getElementById('message').textContent = '❌ 验证失败';
            }
        }

        // 页面加载时生成验证码
        window.onload = refreshCaptcha;
    </script>
</body>
</html>
```

## 📝 API 说明

### NewRedisCaptchaStore

创建 Redis 验证码存储。

```go
captcha.NewRedisCaptchaStore(redis客户端, "键前缀")
```

**示例：**
```go
captchaStore := captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")
```

### NewDefaultImageCaptchaService

创建验证码服务。

```go
captcha.NewDefaultImageCaptchaService(存储实例, 配置选项)
```

**配置选项：**
| 参数        | 说明              | 默认值   |
|------------|------------------|---------|
| ExpireTime | 过期时间          | 5 分钟  |
| Length     | 验证码长度        | 4       |
| Width      | 图片宽度（像素）   | 120     |
| Height     | 图片高度（像素）   | 40      |

**示例：**
```go
service := captcha.NewDefaultImageCaptchaService(captchaStore, captcha.CaptchaOption{
    ExpireTime: 5 * time.Minute,
    Length:     4,
    Width:      120,
    Height:     40,
})
```

### GenerateImageCaptcha

生成验证码。

```go
resp, err := service.GenerateImageCaptcha(上下文, 宽度, 高度)
```

**返回数据：**
- `ID`: 验证码 ID（用于验证）
- `ImageBase64`: 图片数据（直接给前端显示）
- `Value`: 验证码值（仅用于测试，不要返回给前端）

**示例：**
```go
resp, err := service.GenerateImageCaptcha(ctx, 0, 0)
```

### VerifyCaptcha

验证验证码。

```go
isValid, err := service.VerifyCaptcha(上下文, 验证码ID, 用户输入)
```

**返回：**
- `true`: 验证成功（验证码会被自动删除）
- `false`: 验证失败

**示例：**
```go
isValid, err := service.VerifyCaptcha(ctx, captchaId, userInput)
```

## ⚠️ 注意事项

1. **不要返回验证码值给前端**
   ```go
   // ❌ 错误
   return resp.Value
   
   // ✅ 正确
   return resp.ID, resp.ImageBase64
   ```

2. **使用有意义的键前缀**
   ```go
   // ✅ 推荐
   captcha.NewRedisCaptchaStore(redisClient, "myapp:captcha:")
   
   // ❌ 不推荐
   captcha.NewRedisCaptchaStore(redisClient, "")
   ```

3. **验证成功后验证码会自动删除**，防止重复使用

4. **前端可以直接使用 ImageBase64**，无需额外处理

## 🔗 完整示例

查看 `captcha/example/` 目录下的示例代码：
- `integration_with_ioc.go` - 与 IOC 集成示例
- `usage_example.go` - 基本使用示例
- `web_example.go` - Web 应用示例
- `frontend_example.html` - 前端完整示例

## 💬 常见问题

**Q: 支持哪些 Redis 模式？**

A: 支持所有实现了 `redis.Cmdable` 接口的 Redis 客户端：
- 单机模式（`*redis.Client`）
- 集群模式（`*redis.ClusterClient`）
- 哨兵模式（`*redis.Ring`）

**Q: 验证码过期后怎么办？**

A: 验证码会自动过期，验证失败时建议刷新验证码。

**Q: 验证成功后还能再次验证吗？**

A: 不能。验证成功后验证码会被自动删除，防止重复使用。

**Q: 如何测试验证码？**

A: 可以在测试中使用 `resp.Value` 字段查看验证码值，但不要在生产环境返回给前端。

---

有问题？查看 `captcha/OPTIMIZATION_RECOMMENDATIONS.md` 获取更多优化建议。
