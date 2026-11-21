package analyzertest

import (
	"errors"
	"testing"
	"time"

	"m3u-merge-astra/astra/analyzer"

	"github.com/stretchr/testify/assert"
)

func TestFakeCheck(t *testing.T) {
	fake := NewFake()
	fake.AddResult("url1", analyzer.Result{Bitrate: 1})
	fake.AddResult("url2", analyzer.Result{Bitrate: 2})
	fake.AddError("url2", errors.New("error 2"))

	result, err := fake.Check(time.Second, 1, "url1")
	assert.Exactly(t, analyzer.Result{Bitrate: 1}, result)
	assert.NoError(t, err)

	result, err = fake.Check(time.Minute, 2, "url2")
	assert.Exactly(t, analyzer.Result{}, result, "should return zero result with error")
	assert.EqualError(t, err, "error 2")

	result, err = fake.Check(time.Hour, 3, "unknown")
	assert.Exactly(t, analyzer.Result{}, result, "should return zero result for unknown URL")
	assert.NoError(t, err)

	expected := []Call{
		{WatchTime: time.Second, MaxAttempts: 1, URL: "url1"},
		{WatchTime: time.Minute, MaxAttempts: 2, URL: "url2"},
		{WatchTime: time.Hour, MaxAttempts: 3, URL: "unknown"},
	}
	assert.Exactly(t, expected, fake.Calls())
}
