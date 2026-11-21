// Package analyzertest provides fake astra analyzer client for tests.
package analyzertest

import (
	"slices"
	"sync"
	"time"

	"m3u-merge-astra/astra/analyzer"
)

var _ analyzer.Analyzer = (*Fake)(nil)

// Call represents arguments of a single Fake.Check call.
type Call struct {
	WatchTime   time.Duration
	MaxAttempts int
	URL         string
}

// Fake is a fake astra analyzer client, safe for concurrent use.
type Fake struct {
	mu      sync.Mutex
	results map[string]analyzer.Result
	errs    map[string]error
	calls   []Call
}

// NewFake returns new fake astra analyzer client, which returns zero result for unknown URLs.
func NewFake() *Fake {
	return &Fake{
		results: map[string]analyzer.Result{},
		errs:    map[string]error{},
	}
}

// AddResult adds new `result` to return when checking `url`.
func (f *Fake) AddResult(url string, result analyzer.Result) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[url] = result
}

// AddError adds `err` to return when checking `url`, overriding its result.
func (f *Fake) AddError(url string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[url] = err
}

// Check records the call and returns fake result or error for `urlToCheck`.
func (f *Fake) Check(watchTime time.Duration, maxAttempts int, urlToCheck string) (analyzer.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, Call{WatchTime: watchTime, MaxAttempts: maxAttempts, URL: urlToCheck})
	if err := f.errs[urlToCheck]; err != nil {
		return analyzer.Result{}, err
	}
	return f.results[urlToCheck], nil
}

// Calls returns arguments of every Check call in order of calls.
func (f *Fake) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}
