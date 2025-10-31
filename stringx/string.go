package stringx

// ToBytes Go 1.22+ 零分配
func ToBytes(val string) []byte {
	return []byte(val)
}

// ToString 始终零分配
func ToString(val []byte) string {
	return string(val)
}
