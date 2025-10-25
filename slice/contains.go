package slice

// Contains 判断 src 里面是否存在 dst
func Contains[T comparable](src []T, dst T) bool {
	return ContainsFunc[T](src, func(src T) bool {
		return src == dst
	})
}

// ContainsFunc 判断 src 里面是否存在 dst
func ContainsFunc[T any](src []T, equal func(src T) bool) bool {
	for _, v := range src {
		if equal(v) {
			return true
		}
	}
	return false
}

// ContainsAny 判断 src 里面是否存在 dst 中的任何一个元素(一个元素包含即返回true)
func ContainsAny[T comparable](src, dst []T) bool {
	srcMap := toMap[T](src)
	for _, v := range dst {
		if _, exist := srcMap[v]; exist {
			return true
		}
	}
	return false
}

// ContainsAll 判断 src 里面是否存在 dst 中的所有元素
func ContainsAll[T comparable](src, dst []T) bool {
	srcMap := toMap[T](src)
	for _, v := range dst {
		if _, exist := srcMap[v]; !exist {
			return false
		}
	}
	return true
}
