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
	//从index位置开始，后面的元素依次往前挪1个位置
	for i := index; i+1 < length; i++ {
		src[i] = src[i+1]
	}
	//去掉最后一个重复元素
	src = src[:length-1]
	return src, res, nil
}
