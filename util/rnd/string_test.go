package rnd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	uppercaseCharset := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseCharset := "abcdefghijklmnopqrstuvwxyz"
	numbersCharset := "0123456789"

	s := String(0, false, false)
	assert.Exactly(t, "", s)

	s = String(20, false, false)
	assert.Len(t, s, 20)
	assert.False(t, strings.ContainsAny(s, uppercaseCharset))
	assert.True(t, strings.ContainsAny(s, lowercaseCharset))
	assert.False(t, strings.ContainsAny(s, numbersCharset))

	s = String(20, false, true)
	assert.Len(t, s, 20)
	assert.False(t, strings.ContainsAny(s, uppercaseCharset))
	assert.True(t, strings.ContainsAny(s, lowercaseCharset))
	assert.True(t, strings.ContainsAny(s, numbersCharset))

	s = String(20, true, false)
	assert.Len(t, s, 20)
	assert.True(t, strings.ContainsAny(s, uppercaseCharset))
	assert.True(t, strings.ContainsAny(s, lowercaseCharset))
	assert.False(t, strings.ContainsAny(s, numbersCharset))

	s = String(20, true, true)
	assert.Len(t, s, 20)
	assert.True(t, strings.ContainsAny(s, uppercaseCharset))
	assert.True(t, strings.ContainsAny(s, lowercaseCharset))
	assert.True(t, strings.ContainsAny(s, numbersCharset))
}
