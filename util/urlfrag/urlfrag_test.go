package urlfrag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrip(t *testing.T) {
	assert.Exactly(t, "http://url", Strip("http://url"))
	assert.Exactly(t, "http://url", Strip("http://url#"))
	assert.Exactly(t, "http://url", Strip("http://url#a&b"))
	assert.Exactly(t, "http://url/a b", Strip("http://url/a b#a"), "should not encode URL")
	assert.Exactly(t, "dvb://", Strip("dvb://#adapter=0&pnr=1"), "should keep URL without host")
	assert.Exactly(t, "http://{bad/url", Strip("http://{bad/url#a"), "should not parse URL")
	assert.Exactly(t, "http://url", Strip("http://url#a#b"), "should strip after the first #")
}

func TestGet(t *testing.T) {
	assert.Exactly(t, "", Get("http://url"))
	assert.Exactly(t, "", Get("http://url#"))
	assert.Exactly(t, "a", Get("http://url#a"))
	assert.Exactly(t, "a&b", Get("http://url#a&b"))
	assert.Exactly(t, "ua=A%26B", Get("http://url#ua=A%26B"), "should not decode fragment")
	assert.Exactly(t, "adapter=0&pnr=1", Get("dvb://#adapter=0&pnr=1"))
	assert.Exactly(t, "a", Get("http://{bad/url#a"), "should not parse URL")
	assert.Exactly(t, "a#b", Get("http://url#a#b"), "should return everything after the first #")
}

func TestEqual(t *testing.T) {
	tests := []struct {
		a, b     string
		withFrag bool
		expected bool
	}{
		{"http://url/1", "http://url/1", true, true},
		{"http://url/1#a", "http://url/1#b", true, false},
		{"http://url/1#a", "http://url/2#a", true, false},
		{"http://url/1", "http://url/1", false, true},
		{"http://url/1#a", "http://url/1#b", false, true},
		{"http://url/1#a", "http://url/1", false, true},
		{"http://url/1#a", "http://url/2#a", false, false},
		{"http://{bad/url/1#a", "http://{bad/url/1#b", false, true},
		{"http://{bad/url/1#a", "http://{bad/url/2#a", false, false},
		{"dvb://#adapter=0", "dvb://#adapter=1", false, true},
		{"http://url/a b#a", "http://url/a%20b#a", false, false},
	}
	for _, test := range tests {
		assert.Exactly(t, test.expected, Equal(test.a, test.b, test.withFrag), "%q, %q, %v", test.a, test.b,
			test.withFrag)
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		u, frag  string
		expected string
		changed  bool
		msg      string
	}{
		{"http://url", "", "http://url", false, ""},
		{"http://url#a", "", "http://url#a", false, ""},
		{"http://url", "a", "http://url#a", true, ""},
		{"http://url", "#a", "http://url#a", true, "should accept leading #"},
		{"http://url#", "a", "http://url#a", true, "should not add second # to empty fragment"},
		{"http://url#a", "#a", "http://url#a", false, ""},
		{"http://url#a", "b", "http://url#a&b", true, "should merge fragments"},
		{"http://url#a&b", "a", "http://url#a&b", false, ""},
		{"http://url#a&b", "b&a", "http://url#a&b", false, "should ignore order of options"},
		{"http://url#a&b", "c", "http://url#a&b&c", true, ""},
		{"http://url#a&b", "c&d", "http://url#a&b&c&d", true, ""},
		{"http://url#a&b", "#c&d", "http://url#a&b&c&d", true, ""},
		{"http://url#a&b", "a&c", "http://url#a&b&c", true, "should add only missing options"},
		{"http://url#a", "b&b", "http://url#a&b", true, "should not duplicate options"},
		{"http://url#a&&b&", "c", "http://url#a&b&c", true, "should drop empty options on change"},
		{"http://url#ua=A%26B", "no_sync", "http://url#ua=A%26B&no_sync", true, "should keep options encoded"},
		{"dvb://#adapter=0", "pnr=1", "dvb://#adapter=0&pnr=1", true, ""},
		{"http://{bad/url#a", "b", "http://{bad/url#a&b", true, "should not parse URL"},
	}
	for _, test := range tests {
		actual, changed := Add(test.u, test.frag)
		assert.Exactly(t, test.expected, actual, "%q, %q: %s", test.u, test.frag, test.msg)
		assert.Exactly(t, test.changed, changed, "%q, %q: %s", test.u, test.frag, test.msg)
	}
}
