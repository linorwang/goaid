package slice

// DiffSet 差集，只支持 comparable 类型 已去重 并且返回值的顺序是不确定的
func DiffSet[T comparable](src, dst []T) []T {
	srcMap := toMap[T](src)
	for _, val := range dst {
		delete(srcMap, val)
	}

	var ret = make([]T, 0, len(srcMap))
	for key := range srcMap {
		ret = append(ret, key)
	}

	return ret
}
