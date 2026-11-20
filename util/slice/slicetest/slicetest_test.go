package slicetest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithSpare(t *testing.T) {
	s := WithSpare([]int{1, 2}, 9)
	assert.Exactly(t, []int{1, 2}, s)
	assert.Exactly(t, 9, Spare(s))

	_ = append(s, 3, 4)
	assert.Exactly(t, 3, Spare(s), "should be overwritten by append of several elements to the slice")
}
