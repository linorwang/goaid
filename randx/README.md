# randx - 随机字符串生成器

一个简单高效的随机字符串生成工具，支持数字、大小写字母和特殊符号的组合。

## 快速开始

### 导入包

```go
import "github.com/linorwang/goaid/randx"
```

### 基本使用

#### 1. 生成数字验证码（6位）

```go
code, err := randx.RandCode(6, randx.TypeDigit)
if err != nil {
    panic(err)
}
fmt.Println(code) // 输出示例: "482915"
```

#### 2. 生成小写字母（8位）

```go
code, err := randx.RandCode(8, randx.TypeLowerCase)
if err != nil {
    panic(err)
}
fmt.Println(code) // 输出示例: "kjsxmzpq"
```

#### 3. 生成大写字母（10位）

```go
code, err := randx.RandCode(10, randx.TypeUpperCase)
if err != nil {
    panic(err)
}
fmt.Println(code) // 输出示例: "QKXZMNBVPT"
```

#### 4. 生成混合字符（数字+小写+大写，12位）

```go
code, err := randx.RandCode(12, randx.TypeDigit|randx.TypeLowerCase|randx.TypeUpperCase)
if err != nil {
    panic(err)
}
fmt.Println(code) // 输出示例: "a3B9xK2mQ1pL"
```

#### 5. 生成复杂密码（包含特殊符号，16位）

```go
code, err := randx.RandCode(16, randx.TypeMixed)
if err != nil {
    panic(err)
}
fmt.Println(code) // 输出示例: "K9@mX$2pL1&qR#7s"
```

#### 6. 使用自定义字符集

```go
// 定义自己的字符集
customCharset := "ABCDEF123456"
code, err := randx.RandStrByCharset(8, customCharset)
if err != nil {
    panic(err)
}
fmt.Println(code) // 输出示例: "3B1E5D2A"
```

## 类型说明

| 类型 | 值 | 说明 | 包含字符 |
|------|-----|------|----------|
| TypeDigit | 1 | 数字 | 0-9 |
| TypeLowerCase | 2 | 小写字母 | a-z |
| TypeUpperCase | 4 | 大写字母 | A-Z |
| TypeSpecial | 8 | 特殊符号 | ~!@#$%^&*()_+-=[]{};\':\"\\|,./<>? |
| TypeMixed | 15 | 混合类型 | 数字+小写+大写+特殊符号 |

### 类型组合

使用位运算符 `|` 组合多个类型：

```go
// 数字 + 小写字母
typ := randx.TypeDigit | randx.TypeLowerCase

// 大写 + 小写字母
typ := randx.TypeUpperCase | randx.TypeLowerCase

// 全部混合
typ := randx.TypeMixed
```

## API 说明

### RandCode(length int, typ Type) (string, error)

根据指定的长度和类型生成随机字符串。

**参数：**
- `length`: 生成的字符串长度（必须 >= 0）
- `typ`: 字符类型（可组合使用）

**返回：**
- `string`: 生成的随机字符串
- `error`: 错误信息
  - 当 `length < 0` 时返回 `errLengthLessThanZero`
  - 当 `typ` 不在有效范围内时返回 `errTypeNotSupported`

**示例：**

```go
// 生成8位混合字符
code, err := randx.RandCode(8, randx.TypeMixed)
if err != nil {
    log.Fatal(err)
}
fmt.Println(code)
```

### RandStrByCharset(length int, charset string) (string, error)

根据指定的长度和字符集生成随机字符串。

**参数：**
- `length`: 生成的字符串长度（必须 >= 0）
- `charset`: 自定义字符集（不能为空）

**返回：**
- `string`: 生成的随机字符串
- `error`: 错误信息
  - 当 `length < 0` 时返回 `errLengthLessThanZero`
  - 当 `charset` 为空时返回 `errTypeNotSupported`

**示例：**

```go
// 使用自定义字符集
code, err := randx.RandStrByCharset(10, "abcdef0123456789")
if err != nil {
    log.Fatal(err)
}
fmt.Println(code)
```

## 使用场景

### 1. 验证码生成

```go
// 生成6位数字验证码
verifyCode, err := randx.RandCode(6, randx.TypeDigit)
```

### 2. 临时密码生成

```go
// 生成12位强密码（包含特殊符号）
tempPassword, err := randx.RandCode(12, randx.TypeMixed)
```

### 3. 随机文件名生成

```go
// 生成8位小写字母作为文件名
filename, err := randx.RandCode(8, randx.TypeLowerCase)
fullFilename := filename + ".txt"
```

### 4. Session ID 生成

```go
// 生成32位混合字符作为 Session ID
sessionID, err := randx.RandCode(32, randx.TypeMixed)
```

### 5. 优惠券码生成

```go
// 定义优惠券字符集（去除易混淆字符）
couponCharset := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
couponCode, err := randx.RandStrByCharset(10, couponCharset)
```

## 注意事项

⚠️ **安全提示：**
- 此工具使用 `math/rand`，适用于一般场景
- 如需用于密码学安全场景，请使用 `crypto/rand`
- 不要用于生成加密密钥或安全令牌

⚠️ **性能提示：**
- 建议的生成长度：1-1000 字符
- 超长字符串生成可能会消耗较多内存

⚠️ **特殊字符说明：**
- `TypeSpecial` 包含了常见的特殊符号
- 某些特殊字符可能在特定系统中受限
- 可根据需要使用自定义字符集

## 完整示例

```go
package main

import (
	"fmt"
	"log"
	"github.com/linorwang/goaid/randx"
)

func main() {
	// 1. 生成6位数字验证码
	digitCode, err := randx.RandCode(6, randx.TypeDigit)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("数字验证码: %s\n", digitCode)

	// 2. 生成16位强密码
	password, err := randx.RandCode(16, randx.TypeMixed)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("强密码: %s\n", password)

	// 3. 生成Session ID
	sessionID, err := randx.RandCode(32, randx.TypeMixed)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Session ID: %s\n", sessionID)

	// 4. 使用自定义字符集
	customCharset := "ABCDEF0123456789"
	customCode, err := randx.RandStrByCharset(10, customCharset)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("自定义代码: %s\n", customCode)
}
```

## License

MIT License
