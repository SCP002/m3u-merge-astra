package slice

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"m3u-merge-astra/util/logger"

	"github.com/dlclark/regexp2"
	"github.com/stretchr/testify/assert"
)

func TestRxMatchAny(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	rx := regexp2.MustCompile(`^a.*`, regexp2.RE2)
	assert.True(t, RxMatchAny(log, rx, "b", "a0"))
	assert.False(t, RxMatchAny(log, rx, "b", "c"))
}

func TestAnyRxMatch(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	rxList := []*regexp2.Regexp{regexp2.MustCompile(`^a.*`, regexp2.RE2), regexp2.MustCompile(`b`, regexp2.RE2)}
	assert.True(t, AnyRxMatch(log, rxList, "a0"))
	assert.False(t, AnyRxMatch(log, rxList, "c"))
}

func TestAnyRxMatchAny(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	rxList := []*regexp2.Regexp{regexp2.MustCompile(`^a.*`, regexp2.RE2), regexp2.MustCompile(`b`, regexp2.RE2)}
	assert.True(t, AnyRxMatchAny(log, rxList, "a0"))
	assert.True(t, AnyRxMatchAny(log, rxList, "a0", "b"))
	assert.True(t, AnyRxMatchAny(log, rxList, "a0", "c"))
	assert.True(t, AnyRxMatchAny(log, rxList, "b", "c"))

	assert.False(t, AnyRxMatchAny(log, rxList, "c"))
	assert.False(t, AnyRxMatchAny(log, rxList, "c", "d"))
}

// newCatastrophicRx returns regular expression with catastrophic backtracking on input of newCatastrophicInput, which
// times out quickly.
func newCatastrophicRx() *regexp2.Regexp {
	rx := regexp2.MustCompile(`^(a+)+$`, regexp2.RE2)
	rx.MatchTimeout = time.Millisecond * 100
	return rx
}

// newCatastrophicInput returns input which takes too long to match by newCatastrophicRx.
func newCatastrophicInput() string {
	return strings.Repeat("a", 40) + "b"
}

func TestRxMatch(t *testing.T) {
	var out bytes.Buffer
	log := logger.New(logger.DebugLevel, &out)

	rx := regexp2.MustCompile(`^a(?!b)`, regexp2.RE2)
	assert.True(t, RxMatch(log, rx, "ac"))
	assert.False(t, RxMatch(log, rx, "ab"))
	assert.Empty(t, out.String())

	assert.False(t, RxMatch(log, newCatastrophicRx(), newCatastrophicInput()), "should not match on timeout")
	assert.Contains(
		t,
		out.String(),
		`Matching regular expression timed out, considering it not matched: regexp "^(a+)+$"`,
	)
}

func TestRxFindMatch(t *testing.T) {
	var out bytes.Buffer
	log := logger.New(logger.DebugLevel, &out)

	rx := regexp2.MustCompile(`a(\d)`, regexp2.RE2)
	match := RxFindMatch(log, rx, "xa1a2")
	if assert.NotNil(t, match) {
		assert.Exactly(t, "1", match.GroupByNumber(1).String())
	}
	assert.Nil(t, RxFindMatch(log, rx, "x"))
	assert.Empty(t, out.String())

	assert.Nil(t, RxFindMatch(log, newCatastrophicRx(), newCatastrophicInput()), "should not match on timeout")
	assert.Contains(
		t,
		out.String(),
		`Matching regular expression timed out, considering it not matched: regexp "^(a+)+$"`,
	)
}
