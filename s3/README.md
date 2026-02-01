# S3 对象存储客户端

一个全面的 Go 语言 S3 协议兼容对象存储客户端实现，支持上传、删除和获取操作，并返回详细的响应结构，包括便于数据库存储的 URL 拼接。

## 特性

- **S3 协议兼容**: 支持 AWS S3 以及任何 S3 兼容服务（MinIO、Ceph、Wasabi 等）
- **完整的 CRUD 操作**: 支持上传、下载、删除和列出对象
- **URL 拼接**: 自动生成并返回对象 URL，便于存储到数据库
- **元数据支持**: 可以为对象附加自定义元数据
- **批量操作**: 单次请求删除多个对象
- **Context 支持**: 所有操作都支持 Go 的 context，支持取消和超时
- **类型安全响应**: 所有操作都有结构化的返回结果

## 安装

```bash
go get github.com/aws/aws-sdk-go-v2/aws
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/feature/s3/manager
go get github.com/aws/aws-sdk-go-v2/service/s3
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/linorwang/goaid/s3"
)

func main() {
    // 创建 S3 客户端配置
    cfg := &s3.Config{
        Endpoint:        "https://s3.amazonaws.com",
        AccessKeyID:     "your-access-key-id",
        SecretAccessKey: "your-secret-access-key",
        Region:          "us-east-1",
        Bucket:          "your-bucket-name",
        UseSSL:          true,
    }
    
    // 创建 S3 客户端
    client, err := s3.NewClient(cfg)
    if err != nil {
        log.Fatalf("创建 S3 客户端失败: %v", err)
    }
    
    ctx := context.Background()
    
    // 上传文件
    data := []byte("Hello, World!")
    metadata := s3.Metadata{
        "author": "张三",
    }
    
    result, err := client.Upload(ctx, "test/hello.txt", data, "text/plain", metadata)
    if err != nil {
        log.Fatalf("上传失败: %v", err)
    }
    
    fmt.Printf("文件上传成功!\n")
    fmt.Printf("URL: %s\n", result.URL) // 将此 URL 存储到你的数据库
}
```

## 配置说明

`Config` 结构体包含以下字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Endpoint | string | 是 | S3 端点 URL（如 "https://s3.amazonaws.com"） |
| AccessKeyID | string | 是 | 访问密钥 ID |
| SecretAccessKey | string | 是 | 访问密钥 |
| Region | string | 否 | AWS 区域（默认: "us-east-1"） |
| Bucket | string | 是 | 存储桶名称 |
| UseSSL | bool | 否 | 是否使用 SSL（默认: true） |

## API 参考

### Upload

上传对象到 S3 并返回包含 URL 的上传结果。

```go
func (c *Client) Upload(
    ctx context.Context,
    key string,
    data []byte,
    contentType string,
    metadata Metadata,
) (*UploadResult, error)
```

**参数:**
- `ctx`: Go 的 context，用于取消/超时
- `key`: 对象键（存储桶中的路径，如 "images/photo.jpg"）
- `data`: 要上传的字节数据
- `contentType`: 内容类型（如 "image/jpeg"）
- `metadata`: 自定义元数据映射

**返回:** `*UploadResult` 包含:
- `Key`: 对象键
- `URL`: 访问对象的完整 URL
- `ETag`: 实体标签
- `Location`: 完整的位置 URL
- `Size`: 上传对象的大小

**示例:**
```go
data := []byte("文件内容")
metadata := s3.Metadata{
    "uploaded-by": "user123",
    "category": "documents",
}

result, err := client.Upload(ctx, "docs/file.txt", data, "text/plain", metadata)
// 将 result.URL 保存到数据库
```

### UploadFromReader

从 `io.Reader` 上传对象到 S3。

```go
func (c *Client) UploadFromReader(
    ctx context.Context,
    key string,
    reader io.Reader,
    size int64,
    contentType string,
    metadata Metadata,
) (*UploadResult, error)
```

**示例:**
```go
file, _ := os.Open("large_file.mp4")
defer file.Close()
fileInfo, _ := file.Stat()

result, err := client.UploadFromReader(
    ctx, 
    "videos/large_file.mp4", 
    file, 
    fileInfo.Size(), 
    "video/mp4", 
    nil,
)
```

### Get

从 S3 获取对象并返回包含对象数据的获取结果。

```go
func (c *Client) Get(
    ctx context.Context,
    key string,
) (*GetResult, []byte, error)
```

**返回:** `*GetResult` 和 `[]byte` 数据，包含:
- `Key`: 对象键
- `URL`: 访问对象的完整 URL
- `ETag`: 实体标签
- `LastModified`: 最后修改时间戳
- `ContentType`: 内容类型
- `Size`: 对象大小
- `Metadata`: 用户定义的元数据

**示例:**
```go
result, data, err := client.Get(ctx, "images/photo.jpg")
fmt.Printf("下载了 %d 字节\n", len(data))
fmt.Printf("内容类型: %s\n", result.ContentType)
```

### GetObjectInfo

获取对象的元数据而不下载其内容。

```go
func (c *Client) GetObjectInfo(
    ctx context.Context,
    key string,
) (*GetResult, error)
```

**示例:**
```go
info, err := client.GetObjectInfo(ctx, "images/photo.jpg")
fmt.Printf("大小: %d 字节\n", info.Size)
fmt.Printf("最后修改: %s\n", info.LastModified)
```

### GetObjectURL

获取对象的 URL 而不获取对象本身。

```go
func (c *Client) GetObjectURL(key string) string
```

**示例:**
```go
url := client.GetObjectURL("images/photo.jpg")
fmt.Println(url)
```

### Delete

从 S3 删除对象。

```go
func (c *Client) Delete(
    ctx context.Context,
    key string,
) (*DeleteResult, error)
```

**返回:** `*DeleteResult` 包含:
- `Key`: 被删除的对象键
- `Success`: 删除是否成功
- `Message`: 附加消息或错误描述

**示例:**
```go
result, err := client.Delete(ctx, "old_file.txt")
if result.Success {
    fmt.Println("文件删除成功")
}
```

### DeleteMultiple

从 S3 删除多个对象。

```go
func (c *Client) DeleteMultiple(
    ctx context.Context,
    keys []string,
) ([]*DeleteResult, error)
```

**示例:**
```go
keys := []string{"file1.txt", "file2.txt", "file3.txt"}
results, err := client.DeleteMultiple(ctx, keys)

for _, result := range results {
    if result.Success {
        fmt.Printf("已删除: %s\n", result.Key)
    } else {
        fmt.Printf("删除失败 %s: %s\n", result.Key, result.Message)
    }
}
```

### ListObjects

列出存储桶中具有给定前缀的对象。

```go
func (c *Client) ListObjects(
    ctx context.Context,
    prefix string,
) ([]*ObjectInfo, error)
```

**示例:**
```go
objects, err := client.ListObjects(ctx, "images/")
for _, obj := range objects {
    fmt.Printf("%s (%d 字节)\n", obj.Key, obj.Size)
}
```

### Exists

检查对象是否存在于存储桶中。

```go
func (c *Client) Exists(
    ctx context.Context,
    key string,
) (bool, error)
```

**示例:**
```go
exists, err := client.Exists(ctx, "important.txt")
if exists {
    fmt.Println("文件存在")
}
```

## 支持的 S3 兼容服务

此客户端可与任何 S3 兼容服务一起使用:

### AWS S3
```go
cfg := &s3.Config{
    Endpoint:        "https://s3.amazonaws.com",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
    Region:          "us-east-1",
    Bucket:          "your-bucket",
}
```

### MinIO
```go
cfg := &s3.Config{
    Endpoint:        "http://localhost:9000",
    AccessKeyID:     "minioadmin",
    SecretAccessKey: "minioadmin",
    Region:          "us-east-1",
    Bucket:          "my-bucket",
    UseSSL:          false,
}
```

### DigitalOcean Spaces
```go
cfg := &s3.Config{
    Endpoint:        "https://nyc3.digitaloceanspaces.com",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
    Region:          "us-east-1",
    Bucket:          "my-space",
}
```

### Wasabi
```go
cfg := &s3.Config{
    Endpoint:        "https://s3.wasabisys.com",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
    Region:          "us-east-1",
    Bucket:          "my-bucket",
}
```

### 阿里云 OSS
```go
cfg := &s3.Config{
    Endpoint:        "https://oss-cn-hangzhou.aliyuncs.com",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
    Region:          "oss-cn-hangzhou",
    Bucket:          "my-bucket",
}
```

### 腾讯云 COS
```go
cfg := &s3.Config{
    Endpoint:        "https://cos.ap-guangzhou.myqcloud.com",
    AccessKeyID:     "your-secret-id",
    SecretAccessKey: "your-secret-key",
    Region:          "ap-guangzhou",
    Bucket:          "my-bucket-1234567890",
}
```

## 数据库集成

上传结果包含可以直接存储到数据库中的 URL:

```go
// 上传文件
result, err := client.Upload(ctx, "user123/avatar.jpg", imageData, "image/jpeg", nil)

// 存储到数据库
_, err = db.Exec(`
    INSERT INTO user_files (user_id, file_key, file_url, file_size, created_at)
    VALUES (?, ?, ?, ?, NOW())
`, 123, result.Key, result.URL, result.Size)
```

稍后，检索并使用 URL:
```go
var fileURL string
db.QueryRow("SELECT file_url FROM user_files WHERE user_id = ?", 123).Scan(&fileURL)
// fileURL 现在包含访问文件的完整 URL
```

## 错误处理

所有操作都返回可以检查的错误:

```go
result, err := client.Upload(ctx, "file.txt", data, "text/plain", nil)
if err != nil {
    // 处理错误
    log.Printf("上传失败: %v", err)
    return
}
// 使用 result
```

## 示例

请参阅 [example.go](./example.go) 以获取全面的使用示例，包括:
- 基本的上传/下载操作
- 从 reader 上传文件
- 元数据处理
- 批量操作
- 用于数据库存储的 URL 生成
- MinIO 集成

## 许可证

此包是 goaid 项目的一部分。
