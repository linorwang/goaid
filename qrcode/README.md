# qrcode

`qrcode` 是一个与 Web 框架无关的企业级二维码生成包。它使用
`github.com/skip2/go-qrcode` 作为内部编码后端，并提供稳定的公开 API、
PNG/SVG 渲染、Logo、安全限制、结构化内容和原子文件输出。

## 安装

```bash
go get github.com/linorwang/goaid/qrcode
```

## 快速开始

```go
package main

import (
    "os"

    "github.com/linorwang/goaid/qrcode"
)

func main() {
    data, err := qrcode.Generate(
        "https://example.com/orders/1001",
        qrcode.WithSize(512),
        qrcode.WithErrorCorrection(qrcode.ErrorCorrectionMedium),
    )
    if err != nil {
        panic(err)
    }
    if err := os.WriteFile("order.png", data, 0o644); err != nil {
        panic(err)
    }
}
```

`Generate` 默认返回 PNG。直接保存文件时建议使用包内的 `Save`，它通过同目录
临时文件、文件同步和原子提交避免留下半张图片：

```go
err := qrcode.Save(
    "output/order.png",
    "https://example.com/orders/1001",
    qrcode.WithCreateParentDir(true),
    qrcode.WithOverwrite(false),
    qrcode.WithFileMode(0o640),
)
```

## 输出形式

```go
pngBytes, err := qrcode.Generate(content)

svgBytes, err := qrcode.Generate(
    content,
    qrcode.WithFormat(qrcode.FormatSVG),
)

img, err := qrcode.GenerateImage(content)
base64Text, err := qrcode.GenerateBase64(content)
dataURI, err := qrcode.GenerateDataURI(content)
err = qrcode.Write(writer, content)
```

`GenerateImage` 只适用于 PNG 配置；SVG 是矢量文档，不会伪装成
`image.Image`。当前版本不输出 JPEG，避免有损压缩影响扫码可靠性。

## 自定义样式

```go
data, err := qrcode.Generate(
    content,
    qrcode.WithSize(512),
    qrcode.WithMargin(4), // 单位是二维码模块，不是像素
    qrcode.WithForeground(color.RGBA{R: 0x14, G: 0x45, B: 0x2f, A: 0xff}),
    qrcode.WithBackground(color.White),
    qrcode.WithTransparent(false),
    qrcode.WithErrorCorrection(qrcode.ErrorCorrectionQuartile),
)
```

支持四档纠错等级：

- `ErrorCorrectionLow`
- `ErrorCorrectionMedium`（默认）
- `ErrorCorrectionQuartile`
- `ErrorCorrectionHigh`

画布尺寸是精确的输出像素数。内部使用整数宽度绘制每个二维码模块，剩余像素
作为额外外边距居中分配。

## Logo

```go
data, err := qrcode.Generate(
    content,
    qrcode.WithLogoFile("brand.png"),
    qrcode.WithLogoRatio(0.18),
    qrcode.WithLogoPadding(4),
    qrcode.WithLogoCornerRadius(8),
    qrcode.WithLogoBackground(color.White),
)
```

Logo 也可以通过 `WithLogoBytes` 或 `WithLogoImage` 传入。Logo 宽度默认是画布
的 18%，Logo 与底板总宽度不能超过 25%。使用 Logo 时纠错等级会自动提升为
`ErrorCorrectionHigh`。

包不会下载远程 Logo。`WithLogoFile` 只读取调用方明确指定的本地文件，并在完整
解码前检查文件大小和图片尺寸。

## 结构化内容

### Wi-Fi

```go
content, err := qrcode.WIFI(qrcode.WIFIConfig{
    SSID:       "Office-WIFI",
    Password:   "12345678",
    Encryption: qrcode.WPA2,
    Hidden:     false,
})
```

Wi-Fi 标准使用 `WPA` 表示 WPA/WPA2，因此 `WPA2` 会编码为 `T:WPA`。
SSID 和密码中的反斜杠、分号、逗号、冒号和双引号会自动转义。

### 其他类型

```go
urlText, err := qrcode.URL("https://example.com")

vcard, err := qrcode.VCard(qrcode.VCardConfig{
    Name:    "张三",
    Phone:   "13800000000",
    Email:   "zhangsan@example.com",
    Company: "Example Inc.",
})

email, err := qrcode.Email(qrcode.EmailConfig{
    Address: "support@example.com",
    Subject: "问题反馈",
    Body:    "您好，我遇到了一个问题。",
})

phone, err := qrcode.Phone("13800000000")
sms, err := qrcode.SMS("13800000000", "验证码为 123456")
geo, err := qrcode.Geo(39.9042, 116.4074)
```

构造器会校验输入并负责协议转义，不会简单拼接未经检查的字符串。

## 可复用生成器

```go
generator, err := qrcode.NewGenerator(
    qrcode.WithSize(512),
    qrcode.WithMargin(4),
)
if err != nil {
    panic(err)
}

// Generator 创建后配置不可变，可以并发复用。
data, err := generator.Generate("https://example.com")
```

每次生成都会创建独立的底层二维码对象。Logo 在创建生成器时复制，调用方不要在
`NewGenerator` 执行期间并发修改传入的 `image.Image`。

## Context

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

data, err := qrcode.GenerateContext(ctx, content)
err = qrcode.WriteContext(ctx, writer, content)
err = qrcode.SaveContext(ctx, filename, content)
```

Context 会在编码、逐行渲染和文件提交等关键阶段检查。底层编码本身是同步 CPU
计算，已经进入该步骤后不能做到逐指令抢占。

## 默认安全限制

| 限制 | 默认值 |
| --- | ---: |
| 画布尺寸 | 256 px |
| 最大画布尺寸 | 4096 px |
| 静区 | 4 模块 |
| 内容安全上限 | 16 KiB |
| Logo 文件大小 | 5 MiB |
| Logo 图片边长 | 4096 px |
| 输出大小 | 32 MiB |

内容安全上限不等于二维码容量。实际容量取决于内容编码、二维码版本和纠错等级；
超出二维码容量时返回 `ErrContentTooLong`。

限制可以通过 `WithMaxContentBytes`、`WithMaxImageSize`、
`WithMaxLogoBytes`、`WithMaxLogoDimension` 和 `WithMaxOutputBytes` 调整。

## 错误处理

公开错误可以使用 `errors.Is` 判断：

```go
data, err := qrcode.Generate(content)
if errors.Is(err, qrcode.ErrContentTooLong) {
    // 缩短内容或降低纠错等级。
}
```

常用错误包括：

- `ErrEmptyContent`
- `ErrContentTooLong`
- `ErrInvalidSize`
- `ErrInvalidMargin`
- `ErrUnsupportedFormat`
- `ErrInvalidCorrection`
- `ErrInvalidLogo`
- `ErrLogoTooLarge`
- `ErrOutputTooLarge`
- `ErrFileExists`
- `ErrInvalidPayload`

## 测试

```bash
go test ./qrcode
go test -race ./qrcode
go test -fuzz=FuzzGenerate -fuzztime=10s ./qrcode
go test -run '^$' -bench BenchmarkGenerate -benchmem ./qrcode
```

测试使用独立的 ZXing Go 实现重新解码生成结果，覆盖中文、JSON、Emoji、Logo、
并发、文件提交、参数校验、fuzz 和 benchmark。

## 设计边界

本包只负责把合法内容稳定编码成二维码，不负责：

- HTTP Handler 或 Web 框架集成
- 远程 Logo 下载
- 对象存储上传
- 短链接和有效期管理
- 支付状态、鉴权和其他业务生命周期
