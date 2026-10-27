package copier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPDeep(t *testing.T) {
	type Struct struct {
		SliceField []int
	}

	inp := Struct{SliceField: []int{1}}
	assert.Panics(t, func() { MustDeep(&inp) })

	out := MustDeep(inp)
	assert.NotSame(t, &inp, &out)

	inp.SliceField[0] = 10
	assert.Exactly(t, 1, out.SliceField[0], "changes to original value should not modify the copy")
}
