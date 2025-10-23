package slice

import "github.com/linorwang/goaid/internal/errs"

/**
输入参数
src []T：原切片（可空）
index int：删除位置（0 ≤ index < len(src)）
输出
[]T：删除后的新切片（长度 -1，原元素顺序保持）
T：被删除的元素值（如果索引无效，返回零值）
error：如果 index < 0 或 index >= len(src)，返回 errs.NewErrIndexOutOfRange(length, index)；否则 nil
**/

func Delete[T any](src []T, index int) ([]T, T, error) {
	length := len(src)
	if index < 0 || index >= length {
		var zero T
		return nil, zero, errs.NewErrIndexOutOfRange(length, index)
	}
	res := src[index]
	copy(src[index:], src[index+1:]) // 前移
	return src[:length-1], res, nil
}
