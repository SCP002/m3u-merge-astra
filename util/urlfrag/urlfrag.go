// Package urlfrag handles URL fragments (everything after the first '#'), which astra uses for input options such as
// "#no_sync&buffer_time=10".
//
// Fragments are handled as plain text split by '&', without URL decoding, the same way astra does.
package urlfrag

import (
	"strings"

	"github.com/samber/lo"
)

// Strip returns `u` without fragment.
func Strip(u string) string {
	base, _, _ := strings.Cut(u, "#")
	return base
}

// Get returns fragment of `u` without '#', or empty string if `u` has no fragment.
func Get(u string) string {
	_, frag, _ := strings.Cut(u, "#")
	return frag
}

// Equal returns true if `a` equals `b`, ignoring their fragments unless `withFrag` is true.
func Equal(a, b string, withFrag bool) bool {
	if withFrag {
		return a == b
	}
	return Strip(a) == Strip(b)
}

// Add returns `u` with options of `frag` it does not have yet, and true if `u` has been changed.
//
// Leading '#' of `frag` is optional.
func Add(u, frag string) (string, bool) {
	base, currFrag, _ := strings.Cut(u, "#")
	curr := splitOptions(currFrag)
	missing := lo.Uniq(lo.Without(splitOptions(strings.TrimLeft(frag, "#")), curr...))
	if len(missing) == 0 {
		return u, false
	}
	return base + "#" + strings.Join(append(curr, missing...), "&"), true
}

// splitOptions returns non-empty '&' separated options of `frag`.
func splitOptions(frag string) []string {
	return lo.Compact(strings.Split(frag, "&"))
}
