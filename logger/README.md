# Logger

一个基于 Uber Zap 的轻量级日志工具包，提供类型安全的结构化日志功能。

## 特性

- 🔒 类型安全的字段构造
- 🚀 高性能（基于 zap）
- 📦 轻量级包装
- 🎯 简单易用的 API
- 🌳 支持上下文日志
- 📝 默认全局 logger

## 安装

```bash
go get github.com/linorwang/goaid/logger
```

## 快速开始

### 基本使用

```go
package main

import (
    "go.uber.org/zap"
    "github.com/linorwang/goaid/logger"
)

func main() {
    // 创建 zap logger
    zapLogger, _ := zap.NewDevelopment()
    defer zapLogger.Sync()
    
    // 创建我们的 logger 包装器
    log := logger.NewZapLogger(zapLogger)
    
    // 基本日志
    log.Info("server started",
        logger.String("host", "localhost"),
        logger.Int("port", 8080))
}
```

### 使用全局 Logger

```go
import "github.com/linorwang/goaid/logger"

func main() {
    // 使用默认的全局 logger
    logger.Info("application started",
        logger.String("env", "production"))
    
    // 设置自定义全局 logger
    zapLogger, _ := zap.NewDevelopment()
    customLog := logger.NewZapLogger(zapLogger)
    logger.SetDefault(customLog)
}
```

### 错误日志

```go
err := doSomething()
if err != nil {
    log.Error("operation failed",
        logger.Err(err),
        logger.String("operation", "database.connect"))
}
```

### 各种字段类型

```go
log.Debug("debug message",
    logger.String("level", "debug"),
    logger.Int("count", 42),
    logger.Int64("bigNum", 123456789),
    logger.Int32("mediumNum", 12345),
    logger.Float64("ratio", 3.14),
    logger.Bool("enabled", true),
    logger.Time("timestamp", time.Now()),
    logger.Duration("latency", 100*time.Millisecond))
```

### 上下文 Logger

```go
// 创建带有预定义字段的 logger
apiLogger := log.With(
    logger.String("service", "api"),
    logger.String("version", "1.0.0"),
    logger.Int("pid", 1234))

// apiLogger 的所有日志都会包含预定义的字段
apiLogger.Info("request received")
apiLogger.Info("request processed", logger.Int("status", 200))
```

### 结构体日志

```go
user := struct {
    ID       int
    Name     string
    Email    string
}{
    ID:       1,
    Name:     "John Doe",
    Email:    "john@example.com",
}

log.Info("user created", logger.Struct("user", user))
```

## API 文档

### Logger 接口

```go
type Logger interface {
    Debug(msg string, args ...Field)
    Info(msg string, args ...Field)
    Warn(msg string, args ...Field)
    Error(msg string, args ...Field)
    Fatal(msg string, args ...Field)
    Panic(msg string, args ...Field)
    With(args ...Field) Logger
    Sync() error
}
```

### 字段构造函数

| 函数 | 描述 |
|------|------|
| `String(key, val string)` | 字符串字段 |
| `Int(key string, val int)` | 整数字段 |
| `Int64(key string, val int64)` | 64位整数字段 |
| `Int32(key string, val int32)` | 32位整数字段 |
| `Float64(key string, val float64)` | 浮点数字段 |
| `Bool(key string, val bool)` | 布尔字段 |
| `Time(key string, val time.Time)` | 时间字段 |
| `Duration(key string, val time.Duration)` | 时间段字段 |
| `Strs(key string, vals []string)` | 字符串数组字段 |
| `Err(err error)` | 错误字段 |
| `Any(key string, val any)` | 任意类型字段 |
| `Struct(key string, val any)` | 结构体字段 |

### 全局函数

```go
func Debug(msg string, args ...Field)
func Info(msg string, args ...Field)
func Warn(msg string, args ...Field)
func Error(msg string, args ...Field)
func Fatal(msg string, args ...Field)
func Panic(msg string, args ...Field)
func With(args ...Field) Logger
func Sync() error
func SetDefault(l Logger)
```

## 性能优化

本 logger 包通过以下方式优化性能：

1. **类型安全转换**：`toArgs` 方法根据值的实际类型使用对应的 zap 方法，避免使用 `zap.Any()` 的性能损耗

2. **预分配容量**：切片预分配容量，减少内存分配次数

3. **零参数优化**：当没有参数时返回 nil，避免不必要的内存分配

4. **上下文复用**：通过 `With` 方法创建的子 logger 可以复用配置，减少重复设置

## 最佳实践

### 1. 在应用初始化时配置 Logger

```go
func zapConfig() *zap.Logger {
    config := zap.NewProductionConfig()
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zap.RFC3339TimeEncoder
    
    logger, _ := config.Build()
    return logger
}

func main() {
    zapLogger := zapConfig()
    log := logger.NewZapLogger(zapLogger)
    defer log.Sync()
    
    // 使用 log...
}
```

### 2. 使用 With 创建带上下文的 Logger

```go
// 在请求处理器中
func handleRequest(w http.ResponseWriter, r *http.Request) {
    requestLogger := log.With(
        logger.String("request_id", getRequestID(r)),
        logger.String("method", r.Method),
        logger.String("path", r.URL.Path))
    
    requestLogger.Info("request started")
    
    // 处理请求...
    err := processRequest()
    if err != nil {
        requestLogger.Error("request failed", logger.Err(err))
        return
    }
    
    requestLogger.Info("request completed")
}
```

### 3. 记录关键指标

```go
func processWithMetrics() error {
    start := time.Now()
    
    err := doWork()
    
    duration := time.Since(start)
    log.Info("work completed",
        logger.Duration("duration", duration),
        logger.Bool("success", err == nil))
    
    return err
}
```

## 依赖

- [go.uber.org/zap](https://github.com/uber-go/zap) - 高性能 Go 日志库

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](../LICENSE) 文件。
