// Package slicetest provides helpers to test that functions do not append to slices of their arguments.
package slicetest

// spareCap is an amount of spare elements added by WithSpare, enough for appends made in tests.
const spareCap = 16

// WithSpare returns copy of `s` with capacity for more elements, which are set to `spare`.
//
// Appending to the returned slice without copying it overwrites the first of them, which Spare returns.
func WithSpare[T any](s []T, spare T) []T {
	out := make([]T, len(s), len(s)+spareCap)
	copy(out, s)
	for idx := len(s); idx < cap(out); idx++ {
		out[:cap(out)][idx] = spare
	}
	return out
}

// Spare returns the element after the end of `s` made by WithSpare.
func Spare[T any](s []T) T {
	return s[:len(s)+1][len(s)]
}
