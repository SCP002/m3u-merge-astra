package slice

import (
	"m3u-merge-astra/util/copier"
	"m3u-merge-astra/util/slice/slicetest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepend(t *testing.T) {
	assert.Exactly(t, []int{0}, Prepend([]int{}, 0))
	assert.Exactly(t, []int{0, 1}, Prepend([]int{1}, 0))
}

func TestAppendNew(t *testing.T) {
	ol1 := slicetest.WithSpare([]int{0, 1}, -1)
	ol1Original := copier.MustDeep(ol1)

	cbCounter := 0
	ol2 := AppendNew(ol1, func(_ int) {
		cbCounter++
	}, []int{1, 2, 3}...)

	assert.Exactly(t, ol1Original, ol1)
	assert.Exactly(t, -1, slicetest.Spare(ol1), "should not append to `inp`")

	assert.Exactly(t, []int{0, 1, 2, 3}, ol2)
	assert.Exactly(t, 2, cbCounter)

	assert.NotPanics(t, func() { AppendNew(ol1, nil, []int{4}...) }, "should not panic if callback is nil")
}

func TestRemoveLast(t *testing.T) {
	ol1 := []string{"C", "A", "", "A", "B"}
	ol1Original := copier.MustDeep(ol1)

	ol2 := RemoveLast(ol1, "A")
	assert.Exactly(t, ol1Original, ol1)

	expected := []string{"C", "A", "", "B"}
	assert.Exactly(t, expected, ol2, "should remove last object for which predicate returns true")
}

func TestRemoveFirst(t *testing.T) {
	ol1 := []string{"C", "A", "", "A", "B"}
	ol1Original := copier.MustDeep(ol1)

	ol2 := RemoveFirst(ol1, "A")
	assert.Exactly(t, ol1Original, ol1)

	expected := []string{"C", "", "A", "B"}
	assert.Exactly(t, expected, ol2, "should remove first object for which predicate returns true")
}

func TestHasAnyPrefix(t *testing.T) {
	assert.True(t, HasAnyPrefix("some words", "some"))
	assert.False(t, HasAnyPrefix("some words", "words"))
	assert.True(t, HasAnyPrefix("some words", "unknown", "some"))
	assert.False(t, HasAnyPrefix("some words", "unknown", "unknown 2"))
}

func TestIsAllEmpty(t *testing.T) {
	type test struct{}
	assert.True(t, IsAllEmpty([]int{}, []int{}))
	assert.False(t, IsAllEmpty([]string{"something"}, []string{""}))
	assert.True(t, IsAllEmpty(make([]bool, 0), []bool{}))
	assert.False(t, IsAllEmpty([]bool{}, make([]bool, 1)))
	assert.True(t, IsAllEmpty([]test{}, nil))
	assert.False(t, IsAllEmpty(nil, []test{{}}))
}

func TestMapFindDuplBy(t *testing.T) {
	type Struct struct {
		Str string
		Int int
	}

	ol1 := []Struct{{Str: "A", Int: 0}, {Str: "B", Int: 0}, {Str: "A", Int: 0}, {Str: "B", Int: 0}, {Str: "C", Int: 0}}
	ol1Original := copier.MustDeep(ol1)

	// Map to same type.
	ol2Structs := MapFindDuplBy(ol1, func(elm Struct) string {
		return elm.Str
	}, func(elm Struct, idx int, dupl bool) Struct {
		if dupl {
			elm.Int = 1
		}
		return elm
	})
	assert.Exactly(t, ol1Original, ol1)

	expectedStructs := []Struct{
		{Str: "A", Int: 0},
		{Str: "B", Int: 0},
		{Str: "A", Int: 1},
		{Str: "B", Int: 1},
		{Str: "C", Int: 0},
	}
	assert.Exactly(t, expectedStructs, ol2Structs)

	// Map to different type.
	ol2Ints := MapFindDuplBy(ol1, func(elm Struct) string {
		return elm.Str
	}, func(elm Struct, idx int, dupl bool) int {
		if dupl {
			return 1
		}
		return 0
	})
	assert.Exactly(t, ol1Original, ol1)

	expectedInts := []int{0, 0, 1, 1, 0}
	assert.Exactly(t, expectedInts, ol2Ints)
}
