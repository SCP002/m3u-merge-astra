package slice

import (
	"github.com/dlclark/regexp2"
	"github.com/samber/lo"
)

// RxMatch returns true if `rx` matches `elm`.
//
// Error of matching is ignored, as it is returned only on timeout, which is disabled by default.
func RxMatch(rx *regexp2.Regexp, elm string) bool {
	match, _ := rx.MatchString(elm)
	return match
}

// RxMatchAny returns true if `rx` matches any of `elms`.
func RxMatchAny(rx *regexp2.Regexp, elms ...string) bool {
	return lo.ContainsBy(elms, func(elm string) bool {
		return RxMatch(rx, elm)
	})
}

// AnyRxMatch returns true if any element of `rxList` matches `tElm`.
func AnyRxMatch(rxList []*regexp2.Regexp, tElm string) bool {
	return lo.ContainsBy(rxList, func(rx *regexp2.Regexp) bool {
		return RxMatch(rx, tElm)
	})
}

// AnyRxMatchAny returns true if any element of `rxList` matches any of `elms`.
func AnyRxMatchAny(rxList []*regexp2.Regexp, elms ...string) bool {
	return lo.ContainsBy(elms, func(elm string) bool {
		return AnyRxMatch(rxList, elm)
	})
}
