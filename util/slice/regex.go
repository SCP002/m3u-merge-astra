package slice

import (
	"m3u-merge-astra/util/logger"

	"github.com/dlclark/regexp2"
	"github.com/samber/lo"
)

// RxMatch returns true if `rx` matches `elm`.
//
// On timeout of matching it logs error to `log` and returns false.
func RxMatch(log *logger.Logger, rx *regexp2.Regexp, elm string) bool {
	match, err := rx.MatchString(elm)
	if err != nil {
		logTimeout(log, rx, elm)
	}
	return match
}

// RxFindMatch returns first match of `rx` in `elm` or nil if not found.
//
// On timeout of matching it logs error to `log` and returns nil.
func RxFindMatch(log *logger.Logger, rx *regexp2.Regexp, elm string) *regexp2.Match {
	match, err := rx.FindStringMatch(elm)
	if err != nil {
		logTimeout(log, rx, elm)
	}
	return match
}

// RxMatchAny returns true if `rx` matches any of `elms`.
func RxMatchAny(log *logger.Logger, rx *regexp2.Regexp, elms ...string) bool {
	return lo.ContainsBy(elms, func(elm string) bool {
		return RxMatch(log, rx, elm)
	})
}

// AnyRxMatch returns true if any element of `rxList` matches `tElm`.
func AnyRxMatch(log *logger.Logger, rxList []*regexp2.Regexp, tElm string) bool {
	return lo.ContainsBy(rxList, func(rx *regexp2.Regexp) bool {
		return RxMatch(log, rx, tElm)
	})
}

// AnyRxMatchAny returns true if any element of `rxList` matches any of `elms`.
func AnyRxMatchAny(log *logger.Logger, rxList []*regexp2.Regexp, elms ...string) bool {
	return lo.ContainsBy(elms, func(elm string) bool {
		return AnyRxMatch(log, rxList, elm)
	})
}

// logTimeout logs to `log` that matching of `rx` against `elm` timed out.
func logTimeout(log *logger.Logger, rx *regexp2.Regexp, elm string) {
	log.ErrorFi("Matching regular expression timed out, considering it not matched", "regexp", rx.String(),
		"input", elm, "timeout", rx.MatchTimeout)
}
