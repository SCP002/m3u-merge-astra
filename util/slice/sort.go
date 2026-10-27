package slice

import (
	"cmp"
	"slices"

	"m3u-merge-astra/util/copier"
)

// Sort returns deep copy of `list` sorted by Name field.
func Sort[T Named](list []T) (out []T) {
	out = copier.MustDeep(list)

	slices.SortStableFunc(out, func(a, b T) int {
		return cmp.Compare(a.GetName(), b.GetName())
	})

	return
}
