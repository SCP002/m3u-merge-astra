package compare

import (
	"m3u-merge-astra/cfg"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNameSame(t *testing.T) {
	cfg := cfg.General{
		FullTranslitMap:    cfg.DefFullTranslitMap(),
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
	}

	// Test name simplification regex: the + sign.
	assert.False(t, IsNameSame(cfg, "Some Thing (+2)", "@Something2"))

	// Test transliteration and name simplification.
	// Left "НТВ" is a cyrillic visually similar to latin.
	cfg.SimilarTranslit = false
	cfg.FullTranslit = false
	assert.False(t, IsNameSame(cfg, "НТВ HD", "htb hd"))
	assert.False(t, IsNameSame(cfg, "монитор", "Monitor!"))
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"))
	assert.False(t, IsNameSame(cfg, "TV1000 Русское кино", "ТВ 1000 Русское кино"))

	cfg.SimilarTranslit = true
	cfg.FullTranslit = false
	assert.True(t, IsNameSame(cfg, "НТВ HD", "htb hd"))
	assert.False(t, IsNameSame(cfg, "монитор", "Monitor!"))
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"))
	assert.False(t, IsNameSame(cfg, "TV1000 Русское кино", "ТВ 1000 Русское кино"))

	cfg.SimilarTranslit = false
	cfg.FullTranslit = true
	assert.False(t, IsNameSame(cfg, "НТВ HD", "htb hd"))
	assert.True(t, IsNameSame(cfg, "монитор", "Monitor!"))
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"))
	assert.True(t, IsNameSame(cfg, "TV1000 Русское кино", "ТВ 1000 Русское кино"))

	cfg.SimilarTranslit = true
	cfg.FullTranslit = true
	assert.True(t, IsNameSame(cfg, "НТВ HD", "htb hd"))
	assert.True(t, IsNameSame(cfg, "монитор", "Monitor!"))
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"))
	assert.True(t, IsNameSame(cfg, "TV1000 Русское кино", "ТВ 1000 Русское кино"))

	// Test name aliases.
	cfg.NameAliasList = [][]string{
		{"Name 1", "Name 1 var 2", "Name 1 var 3"},
		{"Name 2", "Name 2 var 2"},
		{"Unknown name", "Unknown name var 2", "Unknown name var 3"},
	}
	cfg.SimpleNameAliasList = cfg.SimplifyAliases()

	cfg.NameAliases = false
	msg := "names should not be equvalent as NameAliases = false"
	assert.False(t, IsNameSame(cfg, "Name 1", "Name 1 Var 2"), msg)

	cfg.NameAliases = true
	msg = "names should not be equvalent as transliteration should not apply"
	assert.False(t, IsNameSame(cfg, "Name 1", "Name 1 Вар 2"), msg)

	assert.True(t, IsNameSame(cfg, "Name 1", "name_1_var_2"))
	assert.True(t, IsNameSame(cfg, "name_1_var_3", "Name 1"))

	assert.True(t, IsNameSame(cfg, "Name 2", "name_2_var_2"))

	msg = "names should not be equvalent as such name is absent in name alias list"
	assert.False(t, IsNameSame(cfg, "name_3_var_2", "Name 3"), msg)
}

func TestRemap(t *testing.T) {
	dict := map[string]string{"A": "1", "B": "2", "C": "3"}
	assert.Exactly(t, "123D", remap("ABCD", dict))
}

func TestFirstAlias(t *testing.T) {
	aliases := [][]string{
		{"name1", "name1var2", "name1var3"},
		{"name2", "name2var2"},
		{"unknownname", "unknownnamevar2", "unknownnamevar3"},
	}

	assert.Exactly(t, "name1", firstAlias("name1var2", aliases))
	assert.Exactly(t, "name1", firstAlias("name1var3", aliases))
	assert.Exactly(t, "name1", firstAlias("name1", aliases))

	assert.Exactly(t, "name2", firstAlias("name2var2", aliases))

	assert.Exactly(t, "name3", firstAlias("name3", aliases), "should return input if not found")
}
