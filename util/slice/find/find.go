package find

import (
	"slices"

	"m3u-merge-astra/util/compare"
	"m3u-merge-astra/util/slice"

	"github.com/samber/lo"
)

// IndexOrElse returns unmodified `list`, `list` entry and it's index if `predicate` returns true or `list` with
// `fallback`, `fallback` and it's index (len - 1) if not found. Does not modify `list`.
func IndexOrElse[T any](list []T, fallback T, predicate func(elm T) bool) ([]T, T, int) {
	elm, idx, found := lo.FindIndexOf(list, predicate)
	if !found {
		list = append(slices.Clip(list), fallback)
		return list, fallback, len(list) - 1
	}
	return list, elm, idx
}

// Named returns `list` entry, it's index and true if .Name() of it matching the `name` or empty object, -1 and false if
// not found.
//
// Names are compared with `names`.
func Named[T slice.Named](names *compare.Names, list []T, name string) (T, int, bool) {
	return lo.FindIndexOf(list, func(o T) bool {
		return names.Same(o.GetName(), name)
	})
}

// EverySimilar runs callback `cb` for every entry of `list` starting from index `start` if it's .Name() matching the
// `name`.
//
// Names are compared with `names`.
func EverySimilar[T slice.Named](names *compare.Names, list []T, name string, start int,
	cb func(foundObj T, foundIdx int)) {
	for currIdx := start; currIdx < len(list); currIdx++ {
		currElm := list[currIdx]
		if names.Same(currElm.GetName(), name) {
			cb(currElm, currIdx)
		}
	}
}

// GetSimilar returns entries of `list` whose .Name() matching the `name`.
//
// Names are compared with `names`.
func GetSimilar[T slice.Named](names *compare.Names, list []T, name string) []T {
	return lo.Filter(list, func(elm T, idx int) bool {
		return names.Same(elm.GetName(), name)
	})
}

// HasAnySimilar returns true if `list` contains entry with .Name() matching any of `searchNames`.
//
// Names are compared with `names`.
func HasAnySimilar[T slice.Named](names *compare.Names, list []T, searchNames ...string) bool {
	return lo.ContainsBy(list, func(elm T) bool {
		return lo.SomeBy(searchNames, func(name string) bool {
			return names.Same(elm.GetName(), name)
		})
	})
}
