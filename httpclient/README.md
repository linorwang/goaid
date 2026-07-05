# HTTP Client

一个简单、直接的 Go HTTP 工具包。目标是让调用者少做选择：常规场景用 `Do`，需要原始 `*http.Response` 时再用 `Get`、`Post`、`Send`。

## 安装

```bash
go get github.com/linorwang/goaid/httpclient
```

## 推荐用法：Do

`Do` 会自动读取并关闭响应体，返回增强后的 `*httpclient.Response`。调用者不需要手动 `defer resp.Body.Close()`。

```go
client := httpclient.New()

resp, err := client.Do(
    context.Background(),
    http.MethodGet,
    "https://api.example.com/users",
    httpclient.WithQueryParam("page", "1"),
)
if err != nil {
    return err
}

if !resp.Success() {
    httpErr := resp.Error()
    if httpErr.IsNotFound() {
        // handle 404 here
    }
    return httpErr
}

var users []User
if err := resp.JSON(&users); err != nil {
    return err
}
```

## 发送 JSON

`WithJSON` 会自动序列化请求体，并在未手动设置时添加 `Content-Type: application/json`。

```go
payload := map[string]string{
    "name": "Ada",
}

resp, err := client.Do(
    context.Background(),
    http.MethodPost,
    "https://api.example.com/users",
    httpclient.WithJSON(payload),
    httpclient.WithBearerToken("your-token"),
)
if err != nil {
    return err
}
if !resp.Success() {
    return resp.Error()
}
```

## 需要原始响应时

`Get`、`Post`、`Put`、`Delete`、`Send` 返回标准库的 `*http.Response`。这种模式适合流式下载、自己控制 body 生命周期等场景。

```go
resp, err := client.Get(context.Background(), "https://api.example.com/file")
if err != nil {
    return err
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
```

也可以使用包内辅助函数：

```go
body, err := httpclient.ReadAllResponseBody(resp)
```

## 常用选项

```go
httpclient.WithHeader("X-App", "demo")
httpclient.WithHeaders(map[string]string{"X-App": "demo"})
httpclient.WithQueryParam("page", "1")
httpclient.WithQueryParams(map[string]string{"page": "1"})
httpclient.WithBody([]byte("raw body"))
httpclient.WithJSON(payload)
httpclient.WithBearerToken("token")
httpclient.WithBasicAuth("username", "password")
httpclient.WithTimeout(5 * time.Second)
httpclient.WithRetry(2)
```

## 重试

默认不重试。可以在客户端上设置默认重试次数，也可以在单次请求上覆盖。

```go
client := httpclient.New(
    httpclient.WithDefaultMaxRetries(3),
    httpclient.WithDefaultBackoffStrategy(httpclient.NewExponentialBackoff(
        100*time.Millisecond,
        5*time.Second,
    )),
)

// 使用客户端默认重试
resp, err := client.Do(ctx, http.MethodGet, url)

// 单次请求禁用默认重试
resp, err = client.Do(ctx, http.MethodGet, url, httpclient.WithRetry(0))
```

重试只会自动处理网络错误和 `5xx` 响应；`4xx` 会直接返回给调用者判断。

## 中间件

中间件适合放全局行为，比如日志、认证、User-Agent、请求 ID、指标。

```go
client := httpclient.New()
client.Use(
    httpclient.NewLoggerMiddleware(nil),
    httpclient.NewAuthMiddleware("token"),
    httpclient.NewUserAgentMiddleware("MyApp/1.0"),
)
```

对于单次请求的认证，优先使用 `WithBearerToken` 或 `WithBasicAuth`，调用者更容易看懂当前请求到底带了什么。

## 客户端配置

```go
client := httpclient.New(
    httpclient.WithClientTimeout(30 * time.Second),
    httpclient.WithMaxIdleConns(100),
    httpclient.WithMaxIdleConnsPerHost(20),
    httpclient.WithDefaultMaxRetries(2),
)
```

建议复用同一个 client，不要每次请求都创建新 client。

## 返回值约定

- `Do`：自动读取 body，返回 `*httpclient.Response`，适合大多数 API 调用。
- `Get/Post/Put/Delete/Send`：返回原始 `*http.Response`，调用者负责关闭 body。
- HTTP `4xx/5xx` 不会作为 `error` 返回；使用 `resp.Success()` 和 `resp.Error()` 判断。
- 网络错误、超时、URL 不合法、JSON 序列化失败会作为 `error` 返回。
