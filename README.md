# goaid - Go 工具库集合

一个实用的 Go 工具库集合，包含多个常用功能模块，帮助你快速构建 Go 应用程序。

[![Go Version](https://img.shields.io/badge/Go-1.25.3+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 📦 安装

```bash
go get -u github.com/linorwang/goaid
```

## 🚀 功能模块概览

### 核心工具

#### [captcha](captcha/README.md) - 图片验证码
基于 Redis 的图片验证码生成和验证工具

**主要功能：**
- Redis 存储支持（单例、集群、哨兵）
- 自定义验证码配置（长度、尺寸、过期时间）
- Base64 图片输出
- 验证成功自动删除

---

#### [httpclient](httpclient/README.md) - HTTP 客户端
高性能 HTTP 客户端

**主要功能：**
- 支持多种 HTTP 方法（GET/POST/PUT/DELETE）
- 自动重试机制
- 中间件支持（日志、认证、请求头）
- 统一错误处理和 JSON 解析
- 连接池优化

---

#### [logger](logger/README.md) - 日志工具
基于 Uber Zap 的轻量级日志封装

**主要功能：**
- 类型安全的字段构造
- 高性能（基于 zap）
- 支持上下文日志
- 默认全局 logger
- 结构体日志支持

---

#### [ratelimit](ratelimit/) - 限流器
基于 Redis 的滑动窗口限流器

**主要功能：**
- Redis 滑动窗口算法
- 精确的流量控制
- Lua 脚本保证原子性

---

### 数据存储与处理

#### [s3](s3/README.md) - S3 对象存储
S3 协议兼容对象存储客户端

**主要功能：**
- S3 协议兼容（AWS S3、MinIO、阿里云 OSS、腾讯云 COS）
- 完整的 CRUD 操作
- 自动生成访问 URL
- 元数据支持
- 批量操作

---

#### [slice](slice/) - 切片工具
常用的切片操作工具集

**主要功能：**
- 添加/删除元素
- 查找/包含判断
- 交集/并集/差集计算
- 聚合操作
- 反转/映射

---

#### [stringx](stringx/) - 字符串工具
字符串处理工具

**主要功能：**
- 常用字符串操作
- 类型转换
- 格式化处理

---

### 辅助工具

#### [randx](randx/README.md) - 随机字符串生成
随机字符串生成工具

**主要功能：**
- 支持数字、大小写字母、特殊符号
- 自定义字符集
- 类型组合（位运算）

---

#### [tuple/pair](tuple/pair/) - 元组
简单的元组数据结构

**主要功能：**
- 键值对存储
- 类型安全

---

#### [resp](resp/) - 统一响应
统一响应格式定义

**主要功能：**
- 标准化响应结构
- 便于 API 响应处理

---

#### [sendsms](sendsms/) - 短信发送
多渠道短信发送工具

**主要功能：**
- 支持阿里云
- 支持腾讯云
- 支持华为云
- 统一的接口封装

---

## 📝 项目结构

```
goaid/
├── captcha/          # 图片验证码模块
├── httpclient/       # HTTP 客户端模块
├── logger/           # 日志工具模块
├── ratelimit/        # 限流器模块
├── s3/               # S3 对象存储模块
├── slice/            # 切片工具模块
├── stringx/          # 字符串工具模块
├── randx/            # 随机字符串生成模块
├── tuple/            # 元组模块
│   └── pair/
├── resp/             # 统一响应模块
├── sendsms/          # 短信发送模块
└── internal/         # 内部工具
    ├── errs/         # 错误处理
    └── slice/        # 内部切片工具
```

## 🛠️ 技术栈

- **Go 1.25.3+**
- **[Uber Zap](https://github.com/uber-go/zap)** - 高性能日志库
- **[Redis](https://redis.io/)** - 缓存和限流
- **[base64Captcha](https://github.com/mojocn/base64Captcha)** - 验证码生成
- **[AWS SDK v2](https://github.com/aws/aws-sdk-go-v2)** - S3 客户端
- **[阿里云 SDK](https://github.com/aliyun/alibaba-cloud-sdk-go)** - 阿里云服务
- **[腾讯云 SDK](https://github.com/tencentcloud/tencentcloud-sdk-go)** - 腾讯云服务

## 📚 使用场景

### Web 应用开发
- `httpclient` - 调用第三方 API
- `logger` - 应用日志记录
- `ratelimit` - API 限流保护
- `captcha` - 用户验证码
- `sendsms` - 短信验证码

### 数据处理
- `slice` - 切片操作
- `stringx` - 字符串处理
- `randx` - 随机数据生成
- `resp` - 统一响应格式

### 文件存储
- `s3` - 对象存储（支持 AWS、阿里云、腾讯云等）

### 基础设施
- `logger` - 结构化日志
- `tuple/pair` - 数据结构

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🔗 相关链接

- [GitHub 仓库](https://github.com/linorwang/goaid)
- [提交问题](https://github.com/linorwang/goaid/issues)

---

Made with ❤️ by goaid team
