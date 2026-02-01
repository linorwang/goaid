# HTTP Client

一个简单好用的 Go HTTP 客户端，让发送 HTTP 请求变得超简单！

## 快速开始

### 安装

```bash
go get github.com/linorwang/goaid/httpclient
```

### 第一个请求

```go
package main

import (
    "context"
    "fmt"
    "github.com/linorwang/goaid/httpclient"
)

func main() {
    // 1. 创建客户端
    client := httpclient.New()
    
    // 2. 发送 GET 请求
    resp, err := client.Get(context.Background(), "https://httpbin.org/get")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    
    // 3. 读取响应
    body, _ := httpclient.ReadAllResponseBody(resp)
    fmt.Println(string(body))
}
```

就这么简单！🎉

---

## 基础用法

### GET 请求

```go
// 最简单的 GET 请求
resp, err := client.Get(context.Background(), "https://api.example.com/users")
defer resp.Body.Close()

// 带查询参数
resp, err := client.Get(
    context.Background(),
    "https://api.example.com/users",
    httpclient.WithQueryParam("page", "1"),
    httpclient.WithQueryParam("size", "10"),
)
defer resp.Body.Close()

// 带请求头
resp, err := client.Get(
    context.Background(),
    "https://api.example.com/users",
    httpclient.WithHeader("Authorization", "Bearer your-token"),
)
defer resp.Body.Close()
```

### POST 请求

```go
// 简单的 POST 请求
resp, err := client.Post(context.Background(), "https://api.example.com/users")
defer resp.Body.Close()

// 发送 JSON 数据
resp, err := client.Post(
    context.Background(),
    "https://api.example.com/users",
    httpclient.WithBody([]byte(`{"name": "张三", "email": "zhangsan@example.com"}`)),
    httpclient.WithHeader("Content-Type", "application/json"),
)
defer resp.Body.Close()
```

### PUT 请求

```go
resp, err := client.Put(
    context.Background(),
    "https://api.example.com/users/1",
    httpclient.WithBody([]byte(`{"name": "李四"}`)),
    httpclient.WithHeader("Content-Type", "application/json"),
)
defer resp.Body.Close()
```

### DELETE 请求

```go
resp, err := client.Delete(context.Background(), "https://api.example.com/users/1")
defer resp.Body.Close()
```

---

## 推荐用法：Do 方法

Do 方法更强大，推荐使用！

### 为什么推荐 Do 方法？

- ✅ 自动管理响应体，不需要 `defer resp.Body.Close()`
- ✅ 一行代码判断请求是否成功：`resp.Success()`
- ✅ 自动解析 JSON，不需要手动读取 Body
- ✅ 统一的错误处理

### Do 方法示例

#### GET 请求

```go
resp, err := client.Do(
    context.Background(),
    http.MethodGet,
    "https://api.example.com/users",
    httpclient.WithQueryParam("page", "1"),
)
if err != nil {
    panic(err)
}

if resp.Success() {
    // 自动解析 JSON
    type User struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }
    type Response struct {
        Users []User `json:"users"`
    }
    
    var result Response
    if err := resp.JSON(&result); err != nil {
        panic(err)
    }
    
    fmt.Printf("用户列表: %+v\n", result.Users)
} else {
    fmt.Printf("请求失败: %v\n", resp.Error())
}
```

#### POST 请求（推荐用法）

```go
// 1. 定义请求结构体
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// 2. 定义响应结构体
type UserResponse struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// 3. 准备数据
reqData := CreateUserRequest{
    Name:  "张三",
    Email: "zhangsan@example.com",
}
bodyBytes, _ := json.Marshal(reqData)

// 4. 发送请求
resp, err := client.Do(
    context.Background(),
    http.MethodPost,
    "https://api.example.com/users",
    httpclient.WithBody(bodyBytes),
    httpclient.WithHeader("Content-Type", "application/json"),
)
if err != nil {
    panic(err)
}

// 5. 处理响应
if resp.Success() {
    var result UserResponse
    if err := resp.JSON(&result); err != nil {
        panic(err)
    }
    fmt.Printf("创建成功: ID=%d, Name=%s\n", result.ID, result.Name)
} else {
    fmt.Printf("创建失败: %v\n", resp.Error())
}
```

---

## 自动重试

当请求失败时，自动重试几次，提高成功率。

### 启用自动重试

```go
// 创建客户端，设置最多重试 3 次
client := httpclient.New(
    httpclient.WithDefaultMaxRetries(3),
)

// 发送请求（失败时自动重试）
resp, err := client.Get(context.Background(), "https://api.example.com/data")
```

**重试逻辑**：
- 第 1 次失败 → 等 0.1 秒 → 重试
- 第 2 次失败 → 等 0.2 秒 → 重试
- 第 3 次失败 → 等 0.4 秒 → 重试
- 还失败 → 返回错误

✅ **好处**：网络不稳定时也能成功，不需要关心重试细节。

---

## 错误处理

### 简单判断

```go
resp, err := client.Do(context.Background(), http.MethodGet, url)
if err != nil {
    panic(err)
}

if resp.Success() {
    fmt.Println("请求成功！")
} else {
    fmt.Println("请求失败")
}
```

### 详细错误判断

```go
if !resp.Success() {
    if httpErr := resp.Error(); httpErr != nil {
        // 判断具体错误类型
        if httpErr.IsNotFound() {
            fmt.Println("❌ 资源不存在 (404)")
        } else if httpErr.IsTimeout() {
            fmt.Println("❌ 请求超时")
        } else if httpErr.IsServerError() {
            fmt.Println("❌ 服务器错误 (5xx)")
        } else if httpErr.IsClientError() {
            fmt.Println("❌ 客户端错误 (4xx)")
        }
    }
}
```

---

## 中间件

中间件可以在请求前后做一些额外的事情，比如添加日志、认证等。

### 添加日志中间件

```go
client := httpclient.New()

// 添加日志中间件
client.Use(httpclient.NewLoggerMiddleware(nil))

// 发送请求（会自动打印日志）
resp, err := client.Get(context.Background(), "https://api.example.com/data")
```

### 添加认证中间件

#### Bearer Token 认证（推荐）

```go
client := httpclient.New()

// 使用 Bearer Token
client.Use(httpclient.NewAuthMiddleware("your-api-token"))

// 所有请求都会自动带上 Authorization: Bearer your-api-token
resp, err := client.Get(context.Background(), "https://api.example.com/data")
```

#### Basic Auth 认证

```go
client := httpclient.New()

// 使用 Basic Auth（用户名和密码）
client.Use(httpclient.NewBasicAuthMiddleware("admin", "password123"))

// 所有请求都会自动带上 Authorization: Basic base64(admin:password123)
resp, err := client.Get(context.Background(), "https://api.example.com/data")
```

#### 手动添加 Authorization 头

```go
client := httpclient.New()

// 手动设置 Authorization 头
client.Use(httpclient.NewHeaderMiddleware(map[string]string{
    "Authorization": "Bearer your-custom-token",
}))

resp, err := client.Get(context.Background(), "https://api.example.com/data")
```

### 添加多个中间件

```go
client := httpclient.New()
client.Use(
    httpclient.NewLoggerMiddleware(nil),      // 日志
    httpclient.NewAuthMiddleware("token"),   // 认证
    httpclient.NewUserAgentMiddleware("MyApp/1.0"), // User-Agent
)
```

---

## 客户端配置

### 创建自定义客户端

```go
client := httpclient.New(
    // 超时设置
    httpclient.WithClientTimeout(30 * time.Second),
    
    // 连接池设置
    httpclient.WithMaxIdleConns(100),
    httpclient.WithMaxIdleConnsPerHost(10),
    
    // 自动重试
    httpclient.WithDefaultMaxRetries(3),
)
```

### 常用配置项

| 配置项 | 说明 | 示例 |
|--------|------|------|
| `WithClientTimeout` | 请求超时时间 | `30 * time.Second` |
| `WithMaxIdleConns` | 最大空闲连接数 | `100` |
| `WithDefaultMaxRetries` | 最大重试次数 | `3` |

---

## 完整示例

### 示例 1：调用 API 并解析 JSON

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/linorwang/goaid/httpclient"
)

func main() {
    client := httpclient.New()
    
    // 发送请求
    resp, err := client.Get(context.Background(), "https://jsonplaceholder.typicode.com/users")
    if err != nil {
        panic(err)
    }
    
    // 解析 JSON
    type User struct {
        ID    int    `json:"id"`
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    
    var users []User
    if err := resp.JSON(&users); err != nil {
        panic(err)
    }
    
    // 打印结果
    for _, user := range users {
        fmt.Printf("用户: %s (%d) - %s\n", user.Name, user.ID, user.Email)
    }
}
```

### 示例 2：带重试和日志的生产环境配置

```go
client := httpclient.New(
    // 超时设置
    httpclient.WithClientTimeout(30 * time.Second),
    
    // 连接池设置
    httpclient.WithMaxIdleConns(200),
    httpclient.WithMaxIdleConnsPerHost(100),
    
    // 自动重试（失败时自动重试 3 次）
    httpclient.WithDefaultMaxRetries(3),
)

// 添加中间件
client.Use(
    httpclient.NewLoggerMiddleware(nil),      // 日志
    httpclient.NewAuthMiddleware("your-token"), // 认证
)

// 使用客户端
resp, err := client.Get(context.Background(), "https://api.example.com/data")
if err != nil {
    panic(err)
}

if resp.Success() {
    fmt.Println("请求成功！")
}
```

---

## 方法速查

### 请求方法

```go
// 传统方法（返回 *http.Response）
client.Get(ctx, url, opts...)
client.Post(ctx, url, opts...)
client.Put(ctx, url, opts...)
client.Delete(ctx, url, opts...)

// 推荐方法（返回增强的 *httpclient.Response）
client.Do(ctx, method, url, opts...)
```

### 请求选项

```go
httpclient.WithHeader(key, value)              // 设置请求头
httpclient.WithHeaders(map)                    // 批量设置请求头
httpclient.WithQueryParam(key, value)          // 设置查询参数
httpclient.WithQueryParams(map)                // 批量设置查询参数
httpclient.WithBody(body)                      // 设置请求体
httpclient.WithTimeout(duration)               // 设置超时时间
httpclient.WithRetry(maxRetries)               // 设置重试次数
```

### 响应方法

```go
resp.Success()        // 判断请求是否成功
resp.JSON(&v)         // 解析 JSON
resp.String()         // 获取字符串响应
resp.Bytes()          // 获取字节响应
resp.StatusCode       // 状态码
resp.Error()          // 获取错误
```

---

## 常见问题

### Q: 怎么读取响应内容？

```go
// 方法 1：使用 Do 方法
resp, _ := client.Do(ctx, http.MethodGet, url)
body := resp.String()  // 自动读取

// 方法 2：使用传统方法
resp, _ := client.Get(ctx, url)
body, _ := httpclient.ReadAllResponseBody(resp)
```

### Q: 怎么解析 JSON？

```go
resp, _ := client.Do(ctx, http.MethodGet, url)

type Result struct {
    Name string `json:"name"`
}

var result Result
resp.JSON(&result)  // 自动解析
```

### Q: 怎么处理错误？

```go
resp, err := client.Do(ctx, http.MethodGet, url)
if err != nil {
    // 网络错误、超时等
}

if !resp.Success() {
    // HTTP 错误（404、500 等）
    if resp.Error().IsNotFound() {
        // 404 错误
    }
}
```

---

## 性能优化建议

1. **复用客户端**：不要每次请求都创建新客户端
2. **合理设置超时**：避免连接堆积
3. **启用重试**：网络不稳定时提高成功率
4. **使用中间件**：统一处理日志、认证等

---

## 许可证

MIT License
