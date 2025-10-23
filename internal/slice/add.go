package slice

import "github.com/linorwang/goaid/internal/errs"

/**
输入参数
src []T：原切片。
element T：要插入的元素。
index int：插入位置（0 ≤ index ≤ len(src)）。

输出参数
[]T 结果切片
error：如果 index < 0 或 index > len(src)，返回 errs.NewErrIndexOutOfRange(length, index)；否则 nil
**/

// Add 针对常见“单元素插入”场景
func Add[T any](src []T, element T, index int) ([]T, error) {
	length := len(src)
	if index < 0 || index > length {
		return nil, errs.NewErrIndexOutOfRange(length, index)
	}

	// copy 批量挪动, 效率更高
	src = append(src, element)
	copy(src[index+1:], src[index:])

	return src, nil
}
