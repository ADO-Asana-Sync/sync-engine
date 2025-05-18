package testutil

// Ptr returns a pointer to the given value. Useful for tests.
func Ptr[T any](v T) *T {
	return &v
}
