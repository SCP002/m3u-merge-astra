package url

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	// Both good, with hash.
	equal, err := Equal("http://url/1", "http://url/1", true)
	assert.True(t, equal)
	assert.NoError(t, err)

	equal, err = Equal("http://url/1#a", "http://url/1#b", true)
	assert.False(t, equal)
	assert.NoError(t, err)

	equal, err = Equal("http://url/1#a", "http://url/2#a", true)
	assert.False(t, equal)
	assert.NoError(t, err)

	// Both bad, with hash.
	equal, err = Equal("http://{bad/url/1", "http://{bad/url/1", true)
	assert.True(t, equal)
	assert.NoError(t, err)

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/1#b", true)
	assert.False(t, equal)
	assert.NoError(t, err)

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/2#a", true)
	assert.False(t, equal)
	assert.NoError(t, err)

	// Both good, no hash.
	equal, err = Equal("http://url/1", "http://url/1", false)
	assert.True(t, equal)
	assert.NoError(t, err)

	equal, err = Equal("http://url/1#a", "http://url/1#b", false)
	assert.True(t, equal)
	assert.NoError(t, err)

	equal, err = Equal("http://url/1#a", "http://url/2#a", false)
	assert.False(t, equal)
	assert.NoError(t, err)

	// Right bad, no hash.
	equal, err = Equal("http://url/1", "http://{bad/url/1", false)
	assert.False(t, equal)
	assert.Error(t, err)

	equal, err = Equal("http://url/1#a", "http://{bad/url/1#b", false)
	assert.False(t, equal)
	assert.Error(t, err)

	equal, err = Equal("http://url/1#a", "http://{bad/url/2#a", false)
	assert.False(t, equal)
	assert.Error(t, err)

	// Both bad, no hash.
	equal, err = Equal("http://{bad/url/1", "http://{bad/url/1", false)
	assert.True(t, equal)
	assert.Error(t, err)

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/1#b", false)
	assert.False(t, equal)
	assert.Error(t, err)

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/2#a", false)
	assert.False(t, equal)
	assert.Error(t, err)
}

func TestGetHah(t *testing.T) {
	// Good URL's.
	hash, err := GetHash("http://url")
	assert.Empty(t, hash)
	assert.NoError(t, err)

	hash, err = GetHash("http://url#a")
	assert.Exactly(t, "a", hash)
	assert.NoError(t, err)

	hash, err = GetHash("http://url#a&b")
	assert.Exactly(t, "a&b", hash)
	assert.NoError(t, err)

	// Bad URL's.
	hash, err = GetHash("http://{bad/url")
	assert.Empty(t, hash, "should not retun hash from invalid URL")
	assert.Error(t, err)

	hash, err = GetHash("http://{bad/url#a")
	assert.Empty(t, hash, "should not retun hash from invalid URL")
	assert.Error(t, err)

	hash, err = GetHash("http://{bad/url#a&b")
	assert.Empty(t, hash, "should not retun hash from invalid URL")
	assert.Error(t, err)
}

func TestAddHash(t *testing.T) {
	// Good URL's.
	result, changed, err := AddHash("", "http://url")
	assert.Exactly(t, "http://url", result)
	assert.False(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("", "http://url#a")
	assert.Exactly(t, "http://url#a", result)
	assert.False(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("a", "http://url")
	assert.Exactly(t, "http://url#a", result)
	assert.True(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("#a", "http://url#a")
	assert.Exactly(t, "http://url#a", result)
	assert.False(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("b", "http://url#a")
	assert.Exactly(t, "http://url#a&b", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("a", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b", result)
	assert.False(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("c", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b&c", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("c&d", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b&c&d", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("#c&d", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b&c&d", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err)

	// Bad URL's.
	result, changed, err = AddHash("", "http://{bad/url")
	assert.Exactly(t, "http://{bad/url", result)
	assert.False(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("", "http://{bad/url#a")
	assert.Exactly(t, "http://{bad/url#a", result)
	assert.False(t, changed)
	assert.NoError(t, err)

	result, changed, err = AddHash("a", "http://{bad/url")
	assert.Exactly(t, "http://{bad/url", result)
	assert.False(t, changed)
	assert.Error(t, err)

	result, changed, err = AddHash("#a", "http://{bad/url#a")
	assert.Exactly(t, "http://{bad/url#a", result)
	assert.False(t, changed)
	assert.Error(t, err)

	result, changed, err = AddHash("b", "http://{bad/url#a")
	assert.Exactly(t, "http://{bad/url#a", result)
	assert.False(t, changed)
	assert.Error(t, err)

	result, changed, err = AddHash("a", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result)
	assert.False(t, changed)
	assert.Error(t, err)

	result, changed, err = AddHash("c", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result)
	assert.False(t, changed)
	assert.Error(t, err)

	result, changed, err = AddHash("c&d", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result)
	assert.False(t, changed)
	assert.Error(t, err)

	result, changed, err = AddHash("#c&d", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result)
	assert.False(t, changed)
	assert.Error(t, err)
}

func TestHasParameter(t *testing.T) {
	assert.True(t, hasParameter("a", "a"))
	assert.True(t, hasParameter("a", "#a"))
	assert.False(t, hasParameter("b", "a"))
	assert.False(t, hasParameter("b", "#a"))

	assert.True(t, hasParameter("a", "a&b"))
	assert.True(t, hasParameter("b", "#a&b"))
	assert.False(t, hasParameter("c", "a&b"))
	assert.False(t, hasParameter("c", "#a&b"))

	assert.True(t, hasParameter("a&b", "a&b"))
	assert.True(t, hasParameter("a&b", "#a&b"))
	assert.True(t, hasParameter("a&b", "a&b&c"))
	assert.False(t, hasParameter("a&b&c", "a&b"))

	assert.True(t, hasParameter("b&a", "a&b"))
	assert.True(t, hasParameter("b&a", "#a&b"))
	assert.True(t, hasParameter("b&a", "a&b&c"))
	assert.True(t, hasParameter("b&a", "#a&b&c"))

	assert.False(t, hasParameter("a&c", "a&b"))
	assert.False(t, hasParameter("a&c", "#a&b"))
}
