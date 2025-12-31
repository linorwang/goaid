# goaid 一个简单的 Go 工具

- 获取goaid工具包
```bash
 go get -u github.com/linorwang/goaid
```

目前包含以下功能模块：
- `sendsms`: 多渠道短信发送工具，支持阿里云、腾讯云、华为云
- `httpclient`: 高并发HTTP客户端
- `logger`: zap日志封装
- `ratelimit`: Redis滑动窗口限流
- `slice`: 切片操作工具
- `stringx`: 字符串处理工具
- `randx`: 随机数生成工具
- `tuple/pair`: 元组结构工具

## 功能列表

- **httpclient**: 高性能HTTP客户端，支持高并发请求
- **logger**: 基于zap的日志库封装
- **randx**: 随机数生成工具
- **ratelimit**: 基于Redis的滑动窗口限流器
- **resp**: 统一响应格式
- **slice**: 切片操作工具集
- **stringx**: 字符串处理工具
- **tuple/pair**: 元组数据结构
