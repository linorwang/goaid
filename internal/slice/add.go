package slice

import "github.com/linorwang/goaid/internal/errs"

/**
src []T：原切片。
element T：要插入的元素。
index int：插入位置（0 ≤ index ≤ len(src)）。
**/

// Add 针对常见“单元素插入”场景
func Add[T any](src []T, element T, index int) ([]T, error) {
	length := len(src)
	if index < 0 || index > length {
		return nil, errs.NewErrIndexOutOfRange(length, index)
	}

	// 先将src扩展一个元素, 创建 T 的零值（e.g., int 为 0，string 为 ""），用于占位
	var zeroValue T
	src = append(src, zeroValue)

	// 挪动元素（后移）
	for i := len(src) - 1; i > index; i-- {
		if i-1 >= 0 {
			src[i] = src[i-1]
		}
	}
	src[index] = element
	return src, nil
}
