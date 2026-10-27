package slice

import (
	"m3u-merge-astra/util/copier"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	ol1 := []TestNamedStruct{
		{Name: "C"}, {Name: "A"}, {}, {Name: "B"},
	}
	ol1Original := copier.TestDeep(t, ol1)

	ol2 := Sort(ol1)
	assert.NotSame(t, &ol1, &ol2)
	assert.Exactly(t, ol1Original, ol1)

	expected := []TestNamedStruct{{Name: ""}, {Name: "A"}, {Name: "B"}, {Name: "C"}}
	assert.Exactly(t, expected, ol2)
}
