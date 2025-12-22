# HTTP Client

高性能HTTP客户端，专为高并发场景设计。

## 特性

- 支持高并发请求
- 可自定义超时时间
- 支持各种HTTP方法（GET、POST、PUT、DELETE等）
- 支持查询参数和请求头
- 可配置连接池参数
- 支持自定义Transport

## 安装

```bash
go get -u github.com/linorwang/goaid/httpclient
```

## 快速开始

### 创建客户端

```go
import "github.com/linorwang/goaid/httpclient"

// 创建默认客户端
client := httpclient.New()

// 创建自定义配置客户端
client := httpclient.New(
    httpclient.WithClientTimeout(30*time.Second),
    httpclient.WithMaxIdleConns(100),
    httpclient.WithMaxIdleConnsPerHost(10),
)
```

### 发送GET请求

```go
resp, err := client.Get(
    context.Background(), 
    "https://api.example.com/users",
    httpclient.WithQueryParam("page", "1"),
    httpclient.WithHeader("Authorization", "Bearer token"),
)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

body, err := httpclient.ReadAllResponseBody(resp)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(body))
```

### 发送POST请求

```go
resp, err := client.Post(
    context.Background(),
    "https://api.example.com/users",
    httpclient.WithBody([]byte(`{"name": "John", "email": "john@example.com"}`)),
    httpclient.WithHeader("Content-Type", "application/json"),
)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
```

### 自定义方法请求

```go
resp, err := client.Send(
    context.Background(),
    "PATCH",
    "https://api.example.com/users/1",
    httpclient.WithBody([]byte(`{"name": "Updated Name"}`)),
    httpclient.WithHeader("Content-Type", "application/json"),
)
```

## 配置选项

### 客户端配置

- `WithClientTimeout`: 设置客户端全局超时时间
- `WithMaxIdleConns`: 设置最大空闲连接数
- `WithMaxIdleConnsPerHost`: 设置每个主机的最大空闲连接数
- `WithIdleConnTimeout`: 设置空闲连接超时时间
- `WithTransport`: 设置自定义传输层

### 请求配置

- `WithTimeout`: 设置单个请求超时时间
- `WithHeader`: 设置单个请求头
- `WithHeaders`: 批量设置请求头
- `WithQueryParam`: 设置单个查询参数
- `WithQueryParams`: 批量设置查询参数
- `WithBody`: 设置请求体

## 性能优化建议

1. 复用客户端实例而不是每次创建新的实例
2. 根据实际负载调整连接池参数
3. 合理设置超时时间避免连接堆积
4. 在高并发场景中适当增加MaxIdleConns和MaxIdleConnsPerHost