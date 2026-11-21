package find

import (
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/compare"
	"m3u-merge-astra/util/copier"
	"m3u-merge-astra/util/slice"
	"m3u-merge-astra/util/slice/slicetest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexOrElse(t *testing.T) {
	ol1 := slicetest.WithSpare([]string{"A", "B"}, "spare")
	ol1Original := copier.MustDeep(ol1)

	search := "B"
	ol2, o, idx := IndexOrElse(ol1, search, func(o string) bool {
		return o == search
	})

	assert.Exactly(t, ol1Original, ol1)

	assert.Exactly(t, ol1, ol2)
	assert.Exactly(t, search, o)
	assert.Exactly(t, 1, idx)

	search = "C"
	ol2, o, idx = IndexOrElse(ol1, search, func(o string) bool {
		return o == search
	})

	assert.Exactly(t, ol1Original, ol1)
	assert.Exactly(t, "spare", slicetest.Spare(ol1), "should not append to `list`")

	assert.Exactly(t, []string{"A", "B", "C"}, ol2)
	assert.Exactly(t, search, o)
	assert.Exactly(t, 2, idx)
}

func TestNamed(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol := []slice.TestNamedStruct{
		{Name: "Name"}, {Name: "Name 2"}, {Name: "Name 3"}, {Name: "Name 2"},
	}

	o, idx, found := Named(compare.NewNames(cfg), ol, "name2")
	assert.Exactly(t, ol[1], o)
	assert.Exactly(t, 1, idx, "should return index of first found element")
	assert.True(t, found)

	o, idx, found = Named(compare.NewNames(cfg), ol, "name4")
	assert.Exactly(t, slice.TestNamedStruct{}, o, "should return empty object if not found")
	assert.Exactly(t, -1, idx, "should return index -1 if not found")
	assert.False(t, found, "should return false if no object matching the specified name found")
}

func TestEverySimilar(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol := []slice.TestNamedStruct{
		/* 0 */ {Name: "Name"},
		/* 1 */ {Name: "Name 2"}, // <- Searching similar to this starting from index 2.
		/* 2 */ {Name: "Name 3"},
		/* 3 */ {Name: "Name_2"},
		/* 4 */ {Name: "Name_3"},
		/* 5 */ {Name: "Name-2"},
	}

	idxNameMap := map[int]string{}
	EverySimilar(compare.NewNames(cfg), ol, "name2!", 2, func(foundObj slice.TestNamedStruct, foundIdx int) {
		idxNameMap[foundIdx] = foundObj.Name
	})

	expected := map[int]string{3: "Name_2", 5: "Name-2"}
	assert.Exactly(t, expected, idxNameMap)
}

func TestGetSimilar(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol1 := []slice.TestNamedStruct{
		/* 0 */ {Name: "Name"},
		/* 1 */ {Name: "Name 2"}, // <- Searching similar to this.
		/* 2 */ {Name: "Name 3"},
		/* 3 */ {Name: "Name_2"},
		/* 4 */ {Name: "Name_3"},
		/* 5 */ {Name: "Name-2"},
	}
	ol1Original := copier.MustDeep(ol1)
	ol2 := GetSimilar(compare.NewNames(cfg), ol1, "name2!")

	assert.Exactly(t, ol1Original, ol1)

	expected := []slice.TestNamedStruct{{Name: "Name 2"}, {Name: "Name_2"}, {Name: "Name-2"}}
	assert.Exactly(t, expected, ol2)
}

func TestHasAnySimilar(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol := []slice.TestNamedStruct{{Name: "Name"}, {Name: "Name 2"}, {Name: "Name 3"}}

	assert.True(t, HasAnySimilar(compare.NewNames(cfg), ol, "name2!"))
	assert.False(t, HasAnySimilar(compare.NewNames(cfg), ol, "Name 4"))
	assert.True(t, HasAnySimilar(compare.NewNames(cfg), ol, "Name 4", "name2!"), "should find second object")
}
