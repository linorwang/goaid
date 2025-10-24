package goaid

// RealNumber 接口定义了一个类型约束
type RealNumber interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | // 无符号整数（非负数）
	~int | ~int8 | ~int16 | ~int32 | ~int64 |      // 有符号整数（可正可负）
	~float32 | ~float64 // 浮点数（可正可负）
}

type Number interface {
	RealNumber | ~complex64 | ~complex128
}
