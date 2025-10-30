package slice

// IntersectSet 取交集，只支持 comparable 类型 已去重
func IntersectSet[T comparable](src []T, dst []T) []T {
	srcMap := toMap(src)
	var ret = make([]T, 0, len(src))
	for _, v := range dst {
		if _, exist := srcMap[v]; exist {
			ret = append(ret, v)
		}
	}
	return deduplicate[T](ret)
}

// IntersectSetFunc 支持任意类型，已去重
func IntersectSetFunc[T any](src []T, dst []T, equal equalFunc[T]) []T {
	var ret = make([]T, 0, len(src))
	for _, v := range dst {
		if ContainsFunc[T](src, func(t T) bool {
			return equal(t, v)
		}) {
			ret = append(ret, v)
		}
	}
	return deduplicateFunc[T](ret, equal)
}
