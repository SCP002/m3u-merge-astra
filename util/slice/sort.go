package slice

import (
	"cmp"
	"slices"
)

// Sort returns `list` sorted by Name field. Does not modify `list`.
func Sort[T Named](list []T) (out []T) {
	out = slices.Clone(list)

	slices.SortStableFunc(out, func(a, b T) int {
		return cmp.Compare(a.GetName(), b.GetName())
	})

	return
}
