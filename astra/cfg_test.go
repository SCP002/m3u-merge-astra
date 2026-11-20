package astra

import (
	"testing"

	"m3u-merge-astra/util/copier"
	"m3u-merge-astra/util/slice/slicetest"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCategories(t *testing.T) {
	p := newDefProcessor()

	spareGroup := Group{Name: "Spare"}
	spareCat := Category{Name: "Spare"}
	cl1 := slicetest.WithSpare([]Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: slicetest.WithSpare([]Group{{Name: "C"}, {Name: "D"}, {Name: "D"}}, spareGroup)},
	}, spareCat)
	cl1Original := copier.MustDeep(cl1)
	sl1 := []Stream{
		{Groups: map[string]string{"Category 3": "A"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 3": "C"}},
		{Groups: map[string]string{"Category 3": ""}},
		{Groups: map[string]string{"Category 1": ""}},
		{Groups: map[string]string{"Category 1": "B"}},
		{Groups: map[string]string{"Category 2": "D"}},
		{Groups: map[string]string{"Category 2": "C"}},
		{Groups: map[string]string{"Category 2": "B"}},
		{Groups: map[string]string{"Category 2": "A"}},
	}
	sl1Original := copier.MustDeep(sl1)

	cl2 := p.UpdateCategories(cl1, sl1)

	assert.Exactly(t, cl1Original, cl1)
	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, spareCat, slicetest.Spare(cl1), "should not append to `cats`")
	assert.Exactly(t, spareGroup, slicetest.Spare(cl1[1].Groups), "should not append to groups of `cats`")

	expected := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}, {Name: "D"}, {Name: "B"}, {Name: "A"}}},
		{Name: "Category 3", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "C"}}},
	}
	assert.Exactly(t, expected, cl2)

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	cl1 = []Category{}

	sl1 = []Stream{{Groups: map[string]string{"Cat": "Grp"}}}

	_ = p.UpdateCategories(cl1, sl1)
	assert.Contains(t, out.String(), `Updating categories field with: category "Cat", group "Grp"`)
}

func TestChangedCategories(t *testing.T) {
	p := newDefProcessor()

	// Test with changes, categories to remove in the end of cl1.
	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}}, // 0
		{Name: "Category 2", Groups: []Group{{Name: "B"}}}, // 1
		{Name: "Category 3", Groups: []Group{{Name: "C"}}}, // 2
		{Name: "Category 4", Groups: []Group{{Name: "D"}}}, // 3
		{Name: "Category 5", Groups: []Group{{Name: "E"}}}, // 4
	}
	cl1Original := copier.MustDeep(cl1)

	cl2 := []Category{
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}},
		{Name: "Category 4", Groups: []Group{{Name: "D"}}, Remove: true},
		{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}},
		{Name: "Category 5", Remove: true},
		{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}},
		{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}},
	}
	cl2Original := copier.MustDeep(cl2)

	changed := p.ChangedCategories(cl1, cl2)
	assert.Exactly(t, cl1Original, cl1)
	assert.Exactly(t, cl2Original, cl2)

	expected := []lo.Entry[int, Category]{
		{Key: 1, Value: Category{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}}},
		{Key: -1, Value: Category{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 2, Value: Category{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}}},
		{Key: -1, Value: Category{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 4, Value: Category{Name: "Category 5", Remove: true}},
		{Key: 3, Value: Category{Name: "Category 4", Groups: []Group{{Name: "D"}}, Remove: true}},
	}
	assert.Exactly(t, expected, changed)

	// Test with changes, categories to remove in the beginning of cl1.
	cl1 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}}, // 0
		{Name: "Category 2", Groups: []Group{{Name: "B"}}}, // 1
		{Name: "Category 3", Groups: []Group{{Name: "C"}}}, // 2
		{Name: "Category 4", Groups: []Group{{Name: "D"}}}, // 3
		{Name: "Category 5", Groups: []Group{{Name: "E"}}}, // 4
	}
	cl1Original = copier.MustDeep(cl1)

	cl2 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}, Remove: true},
		{Name: "Category 2", Groups: []Group{{Name: "B"}}, Remove: true},
		{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}},
		{Name: "Category 5", Groups: []Group{{Name: "F"}, {Name: "G"}}},
		{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}},
		{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}},
	}
	cl2Original = copier.MustDeep(cl2)

	changed = p.ChangedCategories(cl1, cl2)
	assert.Exactly(t, cl1Original, cl1)
	assert.Exactly(t, cl2Original, cl2)

	expected = []lo.Entry[int, Category]{
		{Key: -1, Value: Category{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 4, Value: Category{Name: "Category 5", Groups: []Group{{Name: "F"}, {Name: "G"}}}},
		{Key: 2, Value: Category{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}}},
		{Key: -1, Value: Category{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 1, Value: Category{Name: "Category 2", Groups: []Group{{Name: "B"}}, Remove: true}},
		{Key: 0, Value: Category{Name: "Category 1", Groups: []Group{{Name: "A"}}, Remove: true}},
	}
	assert.Exactly(t, expected, changed)

	// Test without changes.
	cl1 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}},
		{Name: "Category 2", Groups: []Group{{Name: "B"}}},
	}
	cl1Original = copier.MustDeep(cl1)

	cl2 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}},
		{Name: "Category 2", Groups: []Group{{Name: "B"}}},
	}
	cl2Original = copier.MustDeep(cl2)

	changed = p.ChangedCategories(cl1, cl2)
	assert.Exactly(t, cl1Original, cl1)
	assert.Exactly(t, cl2Original, cl2)

	expected = nil
	assert.Exactly(t, expected, changed, "should return empty map if no changs found")
}

func TestMergeCategories(t *testing.T) {
	p := newDefProcessor()

	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "B"}}},
		{Name: "Category 1", Groups: []Group{{Name: "C"}, {Name: "A"}, {Name: "D"}}},
		{Name: "Category 2", Groups: []Group{{Name: "A"}, {Name: "C"}}},
		{Name: "Category 2", Groups: []Group{{Name: "D"}, {Name: "E"}}},
		{Name: "Category 3", Groups: []Group{{Name: "X"}}},
	}
	cl1Original := copier.MustDeep(cl1)

	cl2 := p.MergeCategories(cl1)

	assert.Exactly(t, cl1Original, cl1)

	expected := []Category{
		{
			Name:   "Category 1",
			Groups: []Group{{Name: "A"}, {Name: "A", Remove: true}, {Name: "B"}, {Name: "C"}, {Name: "D"}},
		},
		{
			Name:   "Category 2",
			Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "B", Remove: true}, {Name: "C"}, {Name: "D"}, {Name: "E"}},
		},
		{
			Name:   "Category 1",
			Groups: []Group{{Name: "C"}, {Name: "A"}, {Name: "D"}},
			Remove: true,
		},
		{
			Name:   "Category 2",
			Groups: []Group{{Name: "A"}, {Name: "C"}},
			Remove: true,
		},
		{
			Name:   "Category 2",
			Groups: []Group{{Name: "D"}, {Name: "E"}},
			Remove: true,
		},
		{
			Name:   "Category 3",
			Groups: []Group{{Name: "X"}},
		},
	}
	assert.Exactly(t, expected, cl2)
}
