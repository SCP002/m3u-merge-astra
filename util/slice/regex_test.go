package slice

import (
	"testing"

	"github.com/dlclark/regexp2"
	"github.com/stretchr/testify/assert"
)

func TestRxMatchAny(t *testing.T) {
	rx := regexp2.MustCompile(`^a.*`, regexp2.RE2)
	assert.True(t, RxMatchAny(rx, "b", "a0"))
	assert.False(t, RxMatchAny(rx, "b", "c"))
}

func TestAnyRxMatch(t *testing.T) {
	rxList := []*regexp2.Regexp{regexp2.MustCompile(`^a.*`, regexp2.RE2), regexp2.MustCompile(`b`, regexp2.RE2)}
	assert.True(t, AnyRxMatch(rxList, "a0"))
	assert.False(t, AnyRxMatch(rxList, "c"))
}

func TestAnyRxMatchAny(t *testing.T) {
	rxList := []*regexp2.Regexp{regexp2.MustCompile(`^a.*`, regexp2.RE2), regexp2.MustCompile(`b`, regexp2.RE2)}
	assert.True(t, AnyRxMatchAny(rxList, "a0"))
	assert.True(t, AnyRxMatchAny(rxList, "a0", "b"))
	assert.True(t, AnyRxMatchAny(rxList, "a0", "c"))
	assert.True(t, AnyRxMatchAny(rxList, "b", "c"))

	assert.False(t, AnyRxMatchAny(rxList, "c"))
	assert.False(t, AnyRxMatchAny(rxList, "c", "d"))
}
