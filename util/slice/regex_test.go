package slice

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRxMatchAny(t *testing.T) {
	rx := *regexp.MustCompile(`^a.*`)
	assert.True(t, RxMatchAny(rx, "b", "a0"))
	assert.False(t, RxMatchAny(rx, "b", "c"))
}

func TestAnyRxMatch(t *testing.T) {
	rxList := []regexp.Regexp{*regexp.MustCompile(`^a.*`), *regexp.MustCompile(`b`)}
	assert.True(t, AnyRxMatch(rxList, "a0"))
	assert.False(t, AnyRxMatch(rxList, "c"))
}

func TestAnyRxMatchAny(t *testing.T) {
	rxList := []regexp.Regexp{*regexp.MustCompile(`^a.*`), *regexp.MustCompile(`b`)}
	assert.True(t, AnyRxMatchAny(rxList, "a0"))
	assert.True(t, AnyRxMatchAny(rxList, "a0", "b"))
	assert.True(t, AnyRxMatchAny(rxList, "a0", "c"))
	assert.True(t, AnyRxMatchAny(rxList, "b", "c"))

	assert.False(t, AnyRxMatchAny(rxList, "c"))
	assert.False(t, AnyRxMatchAny(rxList, "c", "d"))
}
