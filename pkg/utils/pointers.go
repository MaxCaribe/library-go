package utils

func Dereference[T any](ptr *T) T {
	var zero T
	if ptr == nil {
		return zero
	}
	return *ptr
}

func Reference[T any](value T) *T {
	return &value
}
