package internal

// capi:export
func Add(a int32, b int32) (int32, error) {
	return a + b, nil
}

// capi:export
func Minus(a int32, b int32) (int32, error) {
	return a - b, nil
}
