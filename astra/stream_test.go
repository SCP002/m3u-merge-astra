package astra

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"m3u-merge-astra/astra/analyzer"
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/copier"
	"m3u-merge-astra/util/logger"
	"m3u-merge-astra/util/network/nettest"
	"m3u-merge-astra/util/slice/slicetest"

	"github.com/dlclark/regexp2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestNewStream(t *testing.T) {
	cfg := newDefProcessor().cfg.Streams
	s := NewStream(cfg, "0000", "Name", "Group", []string{"http://url"})

	expected := Stream{
		DisabledInputs: make([]string, 0),
		Enabled:        cfg.MakeNewEnabled,
		HTTPKeepActive: strconv.Itoa(cfg.NewKeepActive),
		ID:             "0000",
		Inputs:         []string{"http://url"},
		Name:           "Name",
		Type:           string(cfg.NewType),
		MarkAdded:      true,
	}
	assert.Exactly(t, expected, s)

	cfg.AddGroupsToNew = true
	s = NewStream(cfg, "0000", "Name", "Group", []string{"http://url"})

	expected.Groups = map[string]string{cfg.GroupsCategoryForNew: "Group"}
	assert.Exactly(t, expected, s)
}

func TestFirstGroup(t *testing.T) {
	s := Stream{}
	assert.Empty(t, s.FirstGroup())
	// Many groups make it unlikely to get the right result relying on random iteration order over map.
	s = Stream{Groups: map[string]string{}}
	for i := 1000; i < 2000; i++ {
		s.Groups[fmt.Sprintf("Category %v", i)] = fmt.Sprintf("Group %v", i)
	}
	assert.Exactly(t, "Category 1000: Group 1000", s.FirstGroup())
}

func TestUpdateStreamInput(t *testing.T) {
	p := newDefProcessor()

	p.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: regexp2.MustCompile("update/from", regexp2.RE2), To: regexp2.MustCompile("update/to", regexp2.RE2)},
		{From: regexp2.MustCompile("update/url", regexp2.RE2), To: regexp2.MustCompile("update/url", regexp2.RE2)},
	}
	s1 := Stream{Inputs: []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/1#b", "http://update/url/1#c",
	}}
	s1Original := copier.MustDeep(s1)

	oldInputs := []string{}
	p.cfg.Streams.KeepInputHash = false
	s2, updated := s1.UpdateInput(p.log, p.cfg.Streams, "http://update/to/1", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	assert.Exactly(t, s1Original, s1)

	expected := []string{
		"http://irrelevant/from#a", "http://update/to/1", "http://update/url/1#b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs)
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(p.log, p.cfg.Streams, "http://update/url/1#b", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/1#b", "http://update/url/1#b",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated discarding old hash")
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(p.log, p.cfg.Streams, "http://irrelevant/to", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	assert.Exactly(t, s1, s2, "inputs should not be updated with irrelevant URL")
	assert.False(t, updated, "should return false as it was not updated")

	p.cfg.Streams.KeepInputHash = true
	s2, updated = s1.UpdateInput(p.log, p.cfg.Streams, "http://update/url/2", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	assert.Exactly(t, s1Original, s1)

	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/2#b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated keeping old hash")
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(p.log, p.cfg.Streams, "http://update/url/2#c", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/2#c&b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated, merging hashes")
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(p.log, p.cfg.Streams, "http://irrelevant/to", func(oldInput string) {
		t.Fail()
	})
	assert.Exactly(t, s1, s2, "inputs should not be updated with irrelevant URL")
	assert.False(t, updated, "should return false as it was not updated")

	assert.Exactly(t, []string{
		"http://update/from/1#c",
		"http://update/url/1#c",
		"http://update/url/1#b",
		"http://update/url/1#b",
	}, oldInputs)
}

func TestHasInput(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	s := Stream{Inputs: []string{"http://other/input", "http://known/input#a"}}

	assert.False(t, s.HasInput(log, "http://known/input", true), "should not contain URL without hash")
	assert.True(t, s.HasInput(log, "http://known/input#a", true))

	assert.True(t, s.HasInput(log, "http://known/input", false), "should contain URL without hash")
	assert.True(t, s.HasInput(log, "http://known/input#b", false), "should contain URL with different hashes")

	assert.False(t, s.HasInput(log, "http://foreign/input", true))
	assert.False(t, s.HasInput(log, "http://foreign/input#a", true))

	assert.False(t, s.HasInput(log, "http://foreign/input", false))
	assert.False(t, s.HasInput(log, "http://foreign/input#b", false))

	s = Stream{Inputs: []string{"http://other/input#a", "http:/other/input/2#b"}}

	assert.False(t, s.HasInput(log, "http://foreign/input", true))
	assert.False(t, s.HasInput(log, "http://foreign/input#a", true))

	assert.False(t, s.HasInput(log, "http://foreign/input", false))
	assert.False(t, s.HasInput(log, "http://foreign/input#b", false))
}

func TestAddInput(t *testing.T) {
	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2"}}
	s1Original := copier.MustDeep(s1)

	s2 := s1.AddInput("http://input/3")
	assert.Exactly(t, s1Original, s1)

	expected := []string{"http://input/3", "http://input/1", "http://input/2"}
	assert.Exactly(t, expected, s2.Inputs)
}

func TestKnownInputs(t *testing.T) {
	config := cfg.NewDefCfg().Streams

	config.InputUpdateMap = []cfg.UpdateRecord{
		{From: regexp2.MustCompile("known/input/1", regexp2.RE2)},
		{From: regexp2.MustCompile("known/input/2", regexp2.RE2)},
	}

	s := Stream{Inputs: []string{"http://known/input/2#a", "http://other/input", "http://known/input/1"}}
	ki := s.KnownInputs(config)

	expected := []string{"http://known/input/2#a", "http://known/input/1"}
	assert.Exactly(t, expected, ki)
}

func TestInputsUpdateNote(t *testing.T) {
	cfg := cfg.NewDefCfg().Streams

	cfg.EnableOnInputUpdate = false
	s := Stream{Enabled: false}
	assert.Exactly(t, "Stream is disabled", s.InputsUpdateNote(cfg))
	s = Stream{Enabled: true}
	assert.Exactly(t, "", s.InputsUpdateNote(cfg), "should not return a note if enabled")

	cfg.EnableOnInputUpdate = true
	s = Stream{Enabled: false}
	assert.Exactly(t, "", s.InputsUpdateNote(cfg), "should not return a note as EnableOnInputUpdate = true")
	s = Stream{Enabled: true}
	assert.Exactly(t, "", s.InputsUpdateNote(cfg), "should not return a note as EnableOnInputUpdate = true")
}

func TestEnableStream(t *testing.T) {
	s1 := Stream{Enabled: false, MarkDisabled: true}
	s1Original := copier.MustDeep(s1)

	s2 := s1.Enable()
	assert.Exactly(t, s1Original, s1)

	expected := Stream{Enabled: true, MarkDisabled: false}
	assert.Exactly(t, expected, s2)

	s1 = Stream{Enabled: false, MarkDisabled: false}
	s1Original = copier.MustDeep(s1)

	s2 = s1.Enable()
	assert.Exactly(t, s1Original, s1)

	expected = Stream{Enabled: true, MarkDisabled: false}
	assert.Exactly(t, expected, s2)

	s1 = Stream{Enabled: true, MarkDisabled: true}
	s1Original = copier.MustDeep(s1)

	s2 = s1.Enable()
	assert.Exactly(t, s1Original, s1)

	expected = Stream{Enabled: true, MarkDisabled: false}
	assert.Exactly(t, expected, s2)

	s1 = Stream{Enabled: true, MarkDisabled: false}
	s1Original = copier.MustDeep(s1)

	s2 = s1.Enable()
	assert.Exactly(t, s1Original, s1)

	assert.Exactly(t, s1, s2, "should not change enabled stream with MarkDisabled field set to false")
}

func TestRemoveInputsCb(t *testing.T) {
	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2", "http://input/1"}}
	s1Original := copier.MustDeep(s1)

	count := 0
	s2 := s1.RemoveInputsCb("http://input/1", func() {
		count++
	})
	assert.Exactly(t, s1Original, s1)

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, s2.Inputs)
	assert.Exactly(t, 2, count)
}

func TestRemoveInputs(t *testing.T) {
	s := Stream{}
	s.removeInputs("") // Should not panic. Tested with RemoveInputsCb.
}

func TestDisableStream(t *testing.T) {
	s1 := Stream{Enabled: false, MarkDisabled: false}
	s1Original := copier.MustDeep(s1)

	s2 := s1.disable()
	assert.Exactly(t, s1Original, s1)

	expected := Stream{Enabled: false, MarkDisabled: true}
	assert.Exactly(t, expected, s2)

	s1 = Stream{Enabled: true, MarkDisabled: false}
	s1Original = copier.MustDeep(s1)

	s2 = s1.disable()
	assert.Exactly(t, s1Original, s1)

	expected = Stream{Enabled: false, MarkDisabled: true}
	assert.Exactly(t, expected, s2)

	s1 = Stream{Enabled: false, MarkDisabled: true}

	s2 = s1.disable()

	assert.Exactly(t, s1, s2)

	s1 = Stream{Enabled: true, MarkDisabled: true}

	s2 = s1.disable()

	expected = Stream{Enabled: false, MarkDisabled: true}
	assert.Exactly(t, expected, s2)
}

func TestRemoveDuplicatedInputsByRx(t *testing.T) {
	p := newDefProcessor()

	p.cfg.Streams.RemoveDuplicatedInputsByRxList = []*regexp2.Regexp{
		regexp2.MustCompile(`^.*:\/\/([^#?/]*)`, regexp2.RE2),     // By host.
		regexp2.MustCompile(`^.*:\/\/.*?\/([^#?]*)`, regexp2.RE2), // By path.
	}

	s1 := Stream{Inputs: []string{
		"http://host1/path1",
		"http://host1/path2",
		"http://host1/path3",
		"http://host2/path1",
		"http://host2/path4",
		"http://host3/path5",
		"",
	}}
	s1Original := copier.MustDeep(s1)

	removed := []string{}
	s2 := s1.removeDuplicatedInputsByRx(p.log, p.cfg.Streams, func(input string) {
		removed = append(removed, input)
	})
	assert.Exactly(t, s1Original, s1)

	expected := []string{"http://host1/path1", "http://host3/path5", ""}
	assert.Exactly(t, expected, s2.Inputs, "should remove inputs duplicated by host and path")

	expected = []string{"http://host1/path2", "http://host1/path3", "http://host2/path4", "http://host2/path1"}
	assert.Exactly(t, expected, removed)
}

func TestDisableAllButOneInputByRx(t *testing.T) {
	p := newDefProcessor()

	p.cfg.Streams.DisableAllButOneInputByRxList = []*regexp2.Regexp{
		regexp2.MustCompile(`[#&]no_sync(&|$)`, regexp2.RE2),
		regexp2.MustCompile(`[#&]i_might_stay(&|$)`, regexp2.RE2),
	}

	s1 := Stream{
		Inputs:         []string{"http://input/1#abc", "http://input/1#no_sync&abc", "http://input/1#abc&i_might_stay"},
		DisabledInputs: slicetest.WithSpare([]string{"http://input/1#def"}, "spare"),
	}
	s1Original := copier.MustDeep(s1)

	disabled := []string{}
	s2 := s1.disableAllButOneInputByRx(p.cfg.Streams, func(input string) {
		disabled = append(disabled, input)
	})
	assert.Exactly(t, s1Original, s1)
	assert.Exactly(t, "spare", slicetest.Spare(s1.DisabledInputs), "should not append to disabled inputs of the stream")

	expected := []string{"http://input/1#no_sync&abc"}
	assert.Exactly(t, expected, s2.Inputs,
		"should remove all inputs except the first matching first found regular expression")

	expected = []string{"http://input/1#abc", "http://input/1#abc&i_might_stay"}
	assert.Exactly(t, expected, disabled)

	expected = []string{"http://input/1#def", "http://input/1#abc", "http://input/1#abc&i_might_stay"}
	assert.Exactly(t, expected, s2.DisabledInputs, "should add all normal removed inputs to the list of disabled ones")

	s1 = Stream{
		Inputs: []string{
			"http://input/1#abc&i_might_stay",
			"http://input/1#abc&i_might_stay",
			"http://input/1#abc",
		},
		DisabledInputs: []string{},
	}
	s1Original = copier.MustDeep(s1)

	disabled = []string{}
	s2 = s1.disableAllButOneInputByRx(p.cfg.Streams, func(input string) {
		disabled = append(disabled, input)
	})
	assert.Exactly(t, s1Original, s1)

	expected = []string{"http://input/1#abc&i_might_stay"}
	assert.Exactly(t, expected, s2.Inputs,
		"should remove all inputs except the first matching first found regular expression")

	expected = []string{"http://input/1#abc&i_might_stay", "http://input/1#abc"}
	assert.Exactly(t, expected, disabled)

	expected = []string{"http://input/1#abc&i_might_stay", "http://input/1#abc"}
	assert.Exactly(t, expected, s2.DisabledInputs, "should add all normal removed inputs to the list of disabled ones")
}

func TestRemoveDisabledInputs(t *testing.T) {
	s1 := Stream{
		Inputs:         []string{"http://input/1"},
		DisabledInputs: []string{"http://input/2", "http://input/2"},
	}
	s1Original := copier.MustDeep(s1)

	removed := []string{}
	s2 := s1.removeDisabledInputs(func(input string) {
		removed = append(removed, input)
	})
	assert.Exactly(t, s1Original, s1)

	expected := []string{"http://input/1"}
	assert.Exactly(t, expected, s2.Inputs)

	expected = []string{"http://input/2", "http://input/2"}
	assert.Exactly(t, expected, removed)

	expected = []string{}
	assert.Exactly(t, expected, s2.DisabledInputs)
}

func TestRemoveBlockedInputs(t *testing.T) {
	cfg := cfg.NewDefCfg().Streams

	cfg.InputBlacklist = []*regexp2.Regexp{
		regexp2.MustCompile("input/1", regexp2.RE2),
		regexp2.MustCompile("input/3", regexp2.RE2),
	}

	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2", "http://input/1", "http://input/3"}}
	s1Original := copier.MustDeep(s1)

	removed := []string{}
	s2 := s1.removeBlockedInputs(cfg, func(input string) {
		removed = append(removed, input)
	})

	assert.Exactly(t, s1Original, s1)

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, s2.Inputs)

	expected = []string{"http://input/1", "http://input/1", "http://input/3"}
	assert.Exactly(t, expected, removed)
}

func TestHasNoInputs(t *testing.T) {
	s := Stream{}
	assert.True(t, s.hasNoInputs())

	s = Stream{Inputs: []string{}}
	assert.True(t, s.hasNoInputs())

	s = Stream{Inputs: []string{"http://input"}}
	assert.False(t, s.hasNoInputs())
}

func TestHasPrefix(t *testing.T) {
	prefix := "prefix: "

	s := Stream{Name: "Name"}
	assert.False(t, s.hasPrefix(prefix))

	s = Stream{Name: prefix + "Name"}
	assert.True(t, s.hasPrefix(prefix))

	s = Stream{Name: "Na" + prefix + "me"}
	assert.False(t, s.hasPrefix(prefix))

	s = Stream{}
	assert.False(t, s.hasPrefix(prefix))

	prefix = ""
	s = Stream{Name: "Name"}
	assert.False(t, s.hasPrefix(prefix))
}

func TestSetPrefix(t *testing.T) {
	prefix := "prefix: "

	s1 := Stream{Name: "Name"}
	s1Original := copier.MustDeep(s1)

	s2 := s1.setPrefix(prefix)
	assert.Exactly(t, s1Original, s1)

	expected := Stream{Name: prefix + "Name"}
	assert.Exactly(t, expected, s2)
}

func TestRemovePrefix(t *testing.T) {
	prefix := "prefix: "

	s1 := Stream{Name: "Name"}
	s1Original := copier.MustDeep(s1)

	s2 := s1.removePrefix(prefix)
	assert.Exactly(t, s1Original, s1)

	assert.Exactly(t, s1, s2, "should not modify the stream without prefix")

	s1 = Stream{Name: prefix + "Name"}
	s1Original = copier.MustDeep(s1)

	s2 = s1.removePrefix(prefix)
	assert.Exactly(t, s1Original, s1)

	expected := Stream{Name: "Name"}
	assert.Exactly(t, expected, s2)

	s1 = Stream{Name: "Na" + prefix + "me"}

	s2 = s1.removePrefix(prefix)

	assert.Exactly(t, s1, s2, "should not modify the stream with prefix string in the middle of the name")

	prefix = ""
	s1 = Stream{Name: "Name"}

	s2 = s1.removePrefix(prefix)

	assert.Exactly(t, s1, s2, "should not modify the stream with empty prefix set in config")
}

func TestAnyHasInput(t *testing.T) {
	p := newDefProcessor()

	sl := []Stream{
		{Inputs: []string{"http://other/input", "http:/other/input/2"}},
		{Inputs: []string{"http://other/input", "http://known/input#a"}},
	}
	assert.False(t, HasInput(p.log, sl, "http://known/input", true), "should not contain URL without hash")
	assert.True(t, HasInput(p.log, sl, "http://known/input#a", true))

	assert.True(t, HasInput(p.log, sl, "http://known/input", false), "should contain URL without hash")
	assert.True(t, HasInput(p.log, sl, "http://known/input#b", false), "should contain URL with different hashes")

	assert.False(t, HasInput(p.log, sl, "http://foreign/input", true))
	assert.False(t, HasInput(p.log, sl, "http://foreign/input#a", true))

	assert.False(t, HasInput(p.log, sl, "http://foreign/input", false))
	assert.False(t, HasInput(p.log, sl, "http://foreign/input#b", false))

	sl = []Stream{
		{Inputs: []string{"http://other/input#a", "http:/other/input/2#b"}},
	}
	assert.False(t, HasInput(p.log, sl, "http://foreign/input", true))
	assert.False(t, HasInput(p.log, sl, "http://foreign/input#a", true))

	assert.False(t, HasInput(p.log, sl, "http://foreign/input", false))
	assert.False(t, HasInput(p.log, sl, "http://foreign/input#b", false))
}

func TestRemoveNamePrefixes(t *testing.T) {
	p := newDefProcessor()
	addedPrefix := p.cfg.Streams.AddedPrefix
	disabledPrefix := p.cfg.Streams.DisabledPrefix

	sl1 := []Stream{
		/* 0 */ {Name: "Name 1", MarkAdded: false, MarkDisabled: false},
		/* 1 */ {Name: disabledPrefix + "Name 2", MarkAdded: false, MarkDisabled: true},
		/* 2 */ {Name: disabledPrefix + "Name 3", MarkAdded: false, MarkDisabled: false},
		/* 3 */ {Name: addedPrefix + "Name 4", MarkAdded: false, MarkDisabled: true},
		/* 4 */ {Name: disabledPrefix + addedPrefix + addedPrefix + "Name 5", MarkAdded: true, MarkDisabled: true},
		/* 5 */ {Name: addedPrefix + disabledPrefix + "Name 6", MarkAdded: false, MarkDisabled: false},
		/* 6 */ {Name: "Na" + addedPrefix + disabledPrefix + "me 7", MarkAdded: false, MarkDisabled: false},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.RemoveNamePrefixes(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	assert.Exactly(t, sl1[0], sl2[0], "should not change the stream without prefixes")

	expected := Stream{Name: "Name 2", MarkAdded: false, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[1])

	expected = Stream{Name: "Name 3", MarkAdded: false, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[2])

	expected = Stream{Name: "Name 4", MarkAdded: true, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[3], "should remove added prefix, set MarkAdded to true and ignore MarkDisabled")

	expected = Stream{Name: "Name 5", MarkAdded: true, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[4])

	expected = Stream{Name: "Name 6", MarkAdded: true, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[5])

	assert.Exactly(t, sl1[6], sl2[6], "should not change the stream with prefix strings in the middle of the name")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	addedPrefix = p.cfg.Streams.AddedPrefix

	sl1 = []Stream{{ID: "0", Name: addedPrefix + "Name 1", Groups: map[string]string{"Cat": "Grp"}}}

	_ = p.RemoveNamePrefixes(sl1)
	assert.Contains(t, out.String(), `Temporarily removing name prefix from stream: ID "0", old name `+
		`"_ADDED: Name 1", new name "Name 1", group "Cat: Grp"`)
}

func TestSort(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		{Name: "C"}, {Name: "A"}, {}, {Name: "B"},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.Sort(sl1)
	assert.Exactly(t, sl1Original, sl1)

	expected := []Stream{{Name: ""}, {Name: "A"}, {Name: "B"}, {Name: "C"}}
	assert.Exactly(t, expected, sl2)
}

func TestAllRemoveBlockedInputs(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.InputBlacklist = []*regexp2.Regexp{
		regexp2.MustCompile("input/1", regexp2.RE2),
		regexp2.MustCompile("input/3", regexp2.RE2),
	}

	sl1 := []Stream{
		{Inputs: []string{"http://input/1", "http://input/2", "http://input/1", "http://input/3"}},
		{Inputs: []string{"http://input/1"}},
		{},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.RemoveBlockedInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, sl2[0].Inputs)

	assert.Len(t, sl2[1].Inputs, 0)

	assert.Len(t, sl2[2].Inputs, 0)

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	p.cfg.Streams.InputBlacklist = []*regexp2.Regexp{regexp2.MustCompile("input/1", regexp2.RE2)}

	sl1 = []Stream{
		{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
	}

	_ = p.RemoveBlockedInputs(sl1)
	assert.Contains(t, out.String(), `Removing blocked input from stream: ID "0", name "Name 1", group "Cat: Grp", `+
		`input "http://input/1"`)
}

func TestRemoveDuplicatedInputs(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		{Inputs: []string{"http://input/2", "http://input/1", "http://input/3", "http://input/2", "http://input/3"}},
		{},
		{Inputs: []string{"http://input/4", "http://input/5", "http://input/1"}},
		{Inputs: []string{"http://input/2", "http://input/4", "http://input/6"}},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.RemoveDuplicatedInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []string{"http://input/2", "http://input/1", "http://input/3"}
	assert.Exactly(t, expected, sl2[0].Inputs)

	assert.Exactly(t, sl1[1], sl2[1], "should not modify streams without inputs")

	expected = []string{"http://input/4", "http://input/5"}
	assert.Exactly(t, expected, sl2[2].Inputs, "should remove inputs existing in previous streams")

	expected = []string{"http://input/6"}
	assert.Exactly(t, expected, sl2[3].Inputs, "should remove inputs existing in previous streams")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	p.cfg.Streams.InputBlacklist = []*regexp2.Regexp{regexp2.MustCompile("input/1", regexp2.RE2)}

	sl1 = []Stream{
		{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
		{ID: "1", Name: "Name 2", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
	}

	_ = p.RemoveDuplicatedInputs(sl1)
	assert.Contains(t, out.String(), `Removing duplicated input from stream: ID "1", name "Name 2", group "Cat: Grp", `+
		`input "http://input/1"`)
}

func TestAllRemoveDuplicatedInputsByRx(t *testing.T) {
	p := newDefProcessor()

	p.cfg.Streams.RemoveDuplicatedInputsByRxList = []*regexp2.Regexp{
		regexp2.MustCompile(`^.*:\/\/([^#?/]*)`, regexp2.RE2),     // By host.
		regexp2.MustCompile(`^.*:\/\/.*?\/([^#?]*)`, regexp2.RE2), // By path.
	}

	sl1 := []Stream{
		/* 0 */ {Inputs: []string{"http://host1/path1", "http://host2/path2", "http://host1/path3"}},
		/* 1 */ {Inputs: []string{"http://host1/path1", "http://host1/path2", "http://host2/path1"}},
		/* 2 */ {Inputs: []string{"http://host1/path1", "http://host2/path2"}},
		/* 3 */ {Inputs: []string{"http://host1/path1", "http://host2/path1"}},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.RemoveDuplicatedInputsByRx(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []string{"http://host1/path1", "http://host2/path2"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should remove inputs duplicated by host")

	expected = []string{"http://host1/path1"}
	assert.Exactly(t, expected, sl2[1].Inputs, "should remove inputs duplicated by host and path")

	assert.Exactly(t, sl1[2], sl2[2], "should not modify stream with unique inputs")

	expected = []string{"http://host1/path1"}
	assert.Exactly(t, expected, sl2[1].Inputs, "should remove inputs duplicated by path")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	p.cfg.Streams.RemoveDuplicatedInputsByRxList = []*regexp2.Regexp{
		regexp2.MustCompile(`^.*:\/\/([^#?/]*)`, regexp2.RE2), // By host.
	}

	sl1 = []Stream{
		{
			ID:     "0",
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://input/1", "http://input/1"},
		},
	}

	_ = p.RemoveDuplicatedInputsByRx(sl1)
	assert.Contains(t, out.String(), `Removing duplicated input per stream by regular expressions: ID "0", `+
		`name "Name 1", group "Cat: Grp", input "http://input/1"`)
}

func TestUniteInputs(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		/* 0 */ {Name: "Name", Inputs: []string{"http://input/a"}},
		/* 1 */ {Name: "Name 3", Inputs: []string{"http://input/a"}},
		/* 2 */ {Name: "Name 2", Inputs: []string{"http://input/a"}},
		/* 3 */ {Name: "Name_3", Inputs: []string{"http://input/b", "http://input/a"}},
		/* 4 */ {Name: "Name-3", Inputs: []string{"http://input/c"}},
		/* 5 */ {Name: "Name_2", Inputs: []string{"http://input/b", "http://input/a"}},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.UniteInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	assert.Exactly(t, sl1[0], sl2[0], "should not modify stream without existing duplicate by name")

	expected := Stream{Name: "Name 3", Inputs: []string{"http://input/c", "http://input/b", "http://input/a"}}
	assert.Exactly(t, expected, sl2[1])

	expected = Stream{Name: "Name 2", Inputs: []string{"http://input/b", "http://input/a"}}
	assert.Exactly(t, expected, sl2[2])

	expected = Stream{Name: "Name_3", Inputs: make([]string, 0), DisabledInputs: make([]string, 0)}
	assert.Exactly(t, expected, sl2[3], "should remove inputs from subsequent streams duplicated by name")

	expected = Stream{Name: "Name-3", Inputs: make([]string, 0), DisabledInputs: make([]string, 0)}
	assert.Exactly(t, expected, sl2[4], "should remove inputs from subsequent streams duplicated by name")

	expected = Stream{Name: "Name_2", Inputs: make([]string, 0), DisabledInputs: make([]string, 0)}
	assert.Exactly(t, expected, sl2[5], "should remove inputs from subsequent streams duplicated by name")

	// Test Streams.EnableOnInputUpdate.
	p.cfg.Streams.EnableOnInputUpdate = false
	sl1 = []Stream{
		{Enabled: false, MarkDisabled: true, Name: "Name", Inputs: []string{"http://input/a"}},
		{Enabled: false, MarkDisabled: true, Name: "Name", Inputs: []string{"http://input/b"}},
	}
	sl1Original = copier.MustDeep(sl1)

	sl2 = p.UniteInputs(sl1)

	assert.False(t, sl2[0].Enabled, "stream should stay disabled as EnableOnInputUpdate = false")
	assert.True(t, sl2[0].MarkDisabled, "MarkDisabled should stay true as EnableOnInputUpdate = false")
	assert.False(t, sl2[1].Enabled, "stream should stay disabled as it has no new inputs")
	assert.True(t, sl2[1].MarkDisabled, "MarkDisabled should stay true as it has no new inputs")

	p.cfg.Streams.EnableOnInputUpdate = true

	sl2 = p.UniteInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.True(t, sl2[0].Enabled, "stream should become enabled as EnableOnInputUpdate = true")
	assert.False(t, sl2[0].MarkDisabled, "MarkDisabled should become false as EnableOnInputUpdate = true")
	assert.False(t, sl2[1].Enabled, "stream should stay disabled as it has no new inputs")
	assert.True(t, sl2[1].MarkDisabled, "MarkDisabled should stay true as it has no new inputs")

	sl1 = []Stream{
		{Enabled: false, MarkDisabled: true, Name: "Name", Inputs: []string{"http://input/a"}},
	}

	sl2 = p.UniteInputs(sl1)

	assert.Exactly(t, sl1, sl2, "should stay the same because it was not updated")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.EnableOnInputUpdate = false

	sl1 = []Stream{
		{ID: "0", Enabled: false, Name: "Name 1", Inputs: []string{"http://input/a"}},
		{ID: "1", Enabled: false, Name: "Name_1", Inputs: []string{"http://input/b"}},
	}

	_ = p.UniteInputs(sl1)
	assert.Contains(t, out.String(), `Uniting inputs of streams: from ID "1", from name "Name_1", `+
		`input "http://input/b", to ID "0", to name "Name 1", note "Stream is disabled"`)

	out.Reset()
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.EnableOnInputUpdate = true

	sl1 = []Stream{
		{Enabled: false, ID: "0", Name: "Name 1", Inputs: []string{"http://input/a"}},
		{Enabled: false, ID: "1", Name: "Name_1", Inputs: []string{"http://input/b"}},
	}

	_ = p.UniteInputs(sl1)
	assert.Contains(t, out.String(), `Enabling the stream (uniting inputs of streams, enable_on_input_update is on): `+
		`ID "0", name "Name 1"`)
}

func TestSortInputs(t *testing.T) {
	p := newDefProcessor()

	// Multiple entries.
	p.cfg.Streams.UnknownInputWeight = 25
	p.cfg.Streams.InputWeightToTypeMap = map[int]*regexp2.Regexp{
		20: regexp2.MustCompile(`input/20`, regexp2.RE2),
		30: regexp2.MustCompile(`input/30`, regexp2.RE2),
	}
	sl1 := []Stream{
		{Inputs: []string{"http://other/a", "http://other/b", "http://input/30", "http://input/20"}},
		{},
		{Inputs: []string{"http://other/c", "http://other/d"}},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.SortInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []string{"http://input/20", "http://other/a", "http://other/b", "http://input/30"}
	assert.Exactly(t, expected, sl2[0].Inputs, "inputs should have this order")

	assert.Exactly(t, sl1[1], sl2[1], "should not modify streams without inputs")

	assert.Exactly(t, sl1[2], sl2[2], "should not modify streams with unknown inputs")

	// One entry.
	p.cfg.Streams.UnknownInputWeight = 30
	p.cfg.Streams.InputWeightToTypeMap = map[int]*regexp2.Regexp{
		20: regexp2.MustCompile(`input/20`, regexp2.RE2),
	}
	sl1 = []Stream{
		{Inputs: []string{"http://other/a", "http://other/b", "http://other/c", "http://input/20"}},
		{Inputs: []string{"http://other/d", "http://other/e"}},
	}
	sl1Original = copier.MustDeep(sl1)

	sl2 = p.SortInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected = []string{"http://input/20", "http://other/a", "http://other/b", "http://other/c"}
	assert.Exactly(t, expected, sl2[0].Inputs, "inputs should have this order")

	assert.Exactly(t, sl1[1], sl2[1], "should not modify streams with unknown inputs")

	// Empty map.
	p.cfg.Streams.UnknownInputWeight = 50
	p.cfg.Streams.InputWeightToTypeMap = map[int]*regexp2.Regexp{}
	sl1 = []Stream{
		{Inputs: []string{"http://other/d", "http://other/c", "http://other/b", "http://other/a"}},
		{Inputs: []string{"http://other/f", "http://other/e"}},
	}
	sl1Original = copier.MustDeep(sl1)

	sl2 = p.SortInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Exactly(t, sl1, sl2)
}

// newDeadInputsServers returns test servers for dead inputs checks:
//   - /alive/ responds with 200;
//   - /dead/404/ responds with 404;
//   - /dead/timeout/ hangs until the client gives up.
func newDeadInputsServers(t *testing.T) nettest.Servers {
	mux := http.NewServeMux()
	mux.HandleFunc("/alive/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
	})
	mux.HandleFunc("/dead/404/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/dead/timeout/", func(w http.ResponseWriter, req *http.Request) {
		<-req.Context().Done()
	})
	return nettest.NewServers(t, mux)
}

func TestRemoveDeadInputs(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.InputMaxConns = 100
	p.cfg.Streams.UseAnalyzer = false
	p.cfg.Streams.DeadInputsCheckBlacklist = []*regexp2.Regexp{
		regexp2.MustCompile(`ignore/1`, regexp2.RE2),
		regexp2.MustCompile(`ignore/2`, regexp2.RE2),
	}

	srv := newDeadInputsServers(t)
	httpClient := srv.Client(time.Millisecond * 300)
	analyzerClient := analyzer.NewFake()

	// Check results.
	sl1 := []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"https://streams.test/dead/timeout/1", "https://streams.test/alive/1"},
		},
		{
			Name:   "Name 2",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://streams.test/alive/2", "http://no-such-host.test/1", "http://ignore/2"},
		},
		{
			Name:   "Name 3",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://streams.test/dead/timeout/" + strings.Repeat("x", 40), "https://ignore/1"},
		},
		{
			Name:   "Name 4",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"rtp://skip/1", "rtsp://skip/2", "file:///skip/3.ts", "http://streams.test/dead/404/1"},
		},
		{
			Name:   "Name 5",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://streams.test/dead/404/2", "rtp://skip/1", "http://streams.test/dead/404/2"},
		},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []string{"https://streams.test/alive/1"}
	assert.Exactly(t, expected, sl2[0].Inputs)

	expected = []string{"http://streams.test/alive/2", "http://ignore/2"}
	assert.Exactly(t, expected, sl2[1].Inputs)

	expected = []string{"https://ignore/1"}
	assert.Exactly(t, expected, sl2[2].Inputs)

	expected = []string{"rtp://skip/1", "rtsp://skip/2", "file:///skip/3.ts"}
	assert.Exactly(t, expected, sl2[3].Inputs, "should not remove inputs with unsupported protocols")

	expected = []string{"rtp://skip/1"}
	assert.Exactly(t, expected, sl2[4].Inputs, "should remove every copy of duplicated dead input")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.UseAnalyzer = false

	sl1 = []Stream{{
		ID:     "0",
		Name:   "Name 1",
		Groups: map[string]string{"Cat": "Grp"},
		Inputs: []string{"https://streams.test/dead/timeout/1"}},
	}

	_ = p.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	msg := `Start checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://streams.test/dead/timeout/1"`
	assert.Contains(t, out.String(), msg)
	msg = `Removing dead input from stream: ID "0", name "Name 1", group "Cat: Grp", ` +
		`input "https://streams.test/dead/timeout/1", reason "Timeout"`
	assert.Contains(t, out.String(), msg)
	msg = `End checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://streams.test/dead/timeout/1"`
	assert.Contains(t, out.String(), msg)
}

// TestDisableDeadInputsConcurrently forces every input check to be in flight at the same time, so checks finish in
// arbitrary order.
func TestDisableDeadInputsConcurrently(t *testing.T) {
	sl1 := []Stream{
		{Name: "Name 1", Inputs: []string{"http://a/dead/1", "http://a/alive/1", "http://a/dead/2"}},
		{Name: "Name 2", Inputs: []string{"https://a/dead/3", "https://a/dead/4", "https://a/dead/5"}},
		{Name: "Name 3", Inputs: []string{"http://a/alive/2", "https://a/dead/6"}},
	}
	inputsAmount := getInputsAmount(sl1)

	// Every request waits until all requests arrive, which is possible only if checks run concurrently.
	var arrived atomic.Int32
	allArrived := make(chan struct{})
	mux := http.NewServeMux()
	barrier := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if arrived.Add(1) == int32(inputsAmount) {
				close(allArrived)
			}
			select {
			case <-allArrived:
				w.WriteHeader(status)
			case <-req.Context().Done():
			}
		}
	}
	mux.HandleFunc("/alive/", barrier(200))
	mux.HandleFunc("/dead/", barrier(404))
	srv := nettest.NewServers(t, mux)

	p := newDefProcessor()
	p.cfg.Streams.InputMaxConns = inputsAmount
	p.cfg.Streams.UseAnalyzer = false

	sl1Original := copier.MustDeep(sl1)
	sl2 := p.DisableDeadInputs(srv.Client(time.Second*5), analyzer.NewFake(), sl1)
	assert.Exactly(t, sl1Original, sl1)

	select {
	case <-allArrived:
	default:
		require.FailNow(t, "input checks did not run concurrently", "%v of %v requests arrived", arrived.Load(),
			inputsAmount)
	}

	// Order of disabled inputs does not depend on the order checks finish in.
	require.Len(t, sl2, len(sl1))
	assert.Exactly(t, []string{"http://a/alive/1"}, sl2[0].Inputs)
	assert.Exactly(t, []string{"http://a/dead/1", "http://a/dead/2"}, sl2[0].DisabledInputs)
	assert.Empty(t, sl2[1].Inputs)
	assert.Exactly(t, []string{"https://a/dead/3", "https://a/dead/4", "https://a/dead/5"}, sl2[1].DisabledInputs)
	assert.Exactly(t, []string{"http://a/alive/2"}, sl2[2].Inputs)
	assert.Exactly(t, []string{"https://a/dead/6"}, sl2[2].DisabledInputs)
}

func TestAnalyzerRemoveDeadInputs(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.InputMaxConns = 100
	p.cfg.Streams.UseAnalyzer = true
	p.cfg.Streams.AnalyzerAudioOnlyBitrateThreshold = 100
	p.cfg.Streams.AnalyzerVideoOnlyBitrateThreshold = 200
	p.cfg.Streams.AnalyzerBitrateThreshold = 300
	p.cfg.Streams.AnalyzerCCErrorsThreshold = 10
	p.cfg.Streams.AnalyzerPCRErrorsThreshold = 20
	p.cfg.Streams.AnalyzerPESErrorsThreshold = 30

	p.cfg.Streams.DeadInputsCheckBlacklist = []*regexp2.Regexp{
		regexp2.MustCompile(`ignore/1`, regexp2.RE2),
	}
	sl1 := []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://dead/audio/50", "http://alive/audio/100", "http://dead/cc/15",
				"http://alive/cc/10"},
		},
		{
			Name:   "Name 2",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"https://dead/video/150", "udp://alive/video/210", "https://dead/pcr/25",
				"https://alive/pcr/20"},
		},
		{
			Name:   "Name 3",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"rtp://dead/full/250", "rtsp://alive/full/400", "udp://dead/pes/35", "rtp://alive/pes/30"},
		},
		{
			Name:   "Name 4",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"dvb://0000#pnr=0000&cam=0000", "xxx://unknown/1", "http://ignore/1"},
		},
	}
	sl1Original := copier.MustDeep(sl1)

	analyzerClient := analyzer.NewFake()
	analyzerClient.AddResult("http://dead/audio/50", analyzer.Result{HasAudio: true, Bitrate: 50})
	analyzerClient.AddResult("http://alive/audio/100", analyzer.Result{HasAudio: true, Bitrate: 100})
	analyzerClient.AddResult("http://dead/cc/15", analyzer.Result{CCErrors: 15, Bitrate: 1000})
	analyzerClient.AddResult("http://alive/cc/10", analyzer.Result{CCErrors: 10, Bitrate: 1000})
	analyzerClient.AddResult("https://dead/video/150", analyzer.Result{HasVideo: true, Bitrate: 150})
	analyzerClient.AddResult("udp://alive/video/210", analyzer.Result{HasVideo: true, Bitrate: 210})
	analyzerClient.AddResult("https://dead/pcr/25", analyzer.Result{PCRErrors: 25, Bitrate: 1000})
	analyzerClient.AddResult("https://alive/pcr/20", analyzer.Result{PCRErrors: 20, Bitrate: 1000})
	analyzerClient.AddResult("rtp://dead/full/250", analyzer.Result{HasAudio: true, HasVideo: true, Bitrate: 250})
	analyzerClient.AddResult("rtsp://alive/full/400", analyzer.Result{HasAudio: true, HasVideo: true, Bitrate: 400})
	analyzerClient.AddResult("udp://dead/pes/35", analyzer.Result{PESErrors: 35, Bitrate: 1000})
	analyzerClient.AddResult("rtp://alive/pes/30", analyzer.Result{PESErrors: 30, Bitrate: 1000})

	var httpClient *http.Client
	sl2 := p.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []string{"http://alive/audio/100", "http://alive/cc/10"}
	assert.Exactly(t, expected, sl2[0].Inputs)

	expected = []string{"udp://alive/video/210", "https://alive/pcr/20"}
	assert.Exactly(t, expected, sl2[1].Inputs)

	expected = []string{"rtsp://alive/full/400", "rtp://alive/pes/30"}
	assert.Exactly(t, expected, sl2[2].Inputs)

	assert.Exactly(t, sl1[3].Inputs, sl2[3].Inputs, "should not remove inputs with unsupported protocols or ignored")

	// Test negative errors threshold.
	p.cfg.Streams.AnalyzerCCErrorsThreshold = -1
	p.cfg.Streams.AnalyzerPCRErrorsThreshold = -1
	p.cfg.Streams.AnalyzerPESErrorsThreshold = -1

	sl1 = []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://alive/cc/0", "http://alive/pcr/1", "http://alive/pes/10"},
		},
	}
	sl1Original = copier.MustDeep(sl1)

	analyzerClient.AddResult("http://alive/cc/0", analyzer.Result{CCErrors: 0, Bitrate: 1000})
	analyzerClient.AddResult("http://alive/pcr/1", analyzer.Result{PCRErrors: 1, Bitrate: 1000})
	analyzerClient.AddResult("http://alive/pes/10", analyzer.Result{PESErrors: 10, Bitrate: 1000})

	sl2 = p.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	assert.Exactly(t, sl1[0].Inputs, sl2[0].Inputs, "should not remove inputs with errors if thresholds are negative")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.UseAnalyzer = true
	p.cfg.Streams.AnalyzerAudioOnlyBitrateThreshold = 100

	sl1 = []Stream{{
		ID:     "0",
		Name:   "Name 1",
		Groups: map[string]string{"Cat": "Grp"},
		Inputs: []string{"https://dead/audio/50"}},
	}

	analyzerClient = analyzer.NewFake()
	analyzerClient.AddResult("https://dead/audio/50", analyzer.Result{HasAudio: true, Bitrate: 50})

	_ = p.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	msg := `Start checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://dead/audio/50"`
	assert.Contains(t, out.String(), msg)
	msg = `Removing dead input from stream: ID "0", name "Name 1", group "Cat: Grp", ` +
		`input "https://dead/audio/50", reason "Bitrate 50 < 100"`
	assert.Contains(t, out.String(), msg)
	msg = `End checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://dead/audio/50"`
	assert.Contains(t, out.String(), msg)
}

func TestProgressRemoveDeadInputs(t *testing.T) {
	var out syncBuffer
	p := newDefProcessorWithWriter(&out)
	p.cfg.Streams.InputMaxConns = 1
	p.cfg.Streams.UseAnalyzer = false

	srv := newDeadInputsServers(t)
	httpClient := srv.Client(time.Second * 3)
	analyzerClient := analyzer.NewFake()

	sl1 := []Stream{{Inputs: []string{
		"http://streams.test/alive/1",
		"http://streams.test/dead/404/1",
		"udp://unsupported/1",
		"http://streams.test/dead/404/2",
	}}}

	// Log every check.
	p.progressInterval = 0
	_ = p.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	for _, progress := range []string{"1 / 4 (25%)", "2 / 4 (50%)", "3 / 4 (75%)", "4 / 4 (100%)"} {
		assert.Contains(t, out.String(), `Removing dead inputs from streams: progress "`+progress+`"`)
	}
	assert.Contains(t, out.String(), `Removing dead inputs from streams finished: checked "3", skipped "1", `+
		`removed "2", took "`)

	// Log first check as nothing is logged yet, then wait for interval, but log the last check anyway.
	out.Reset()
	p.progressInterval = time.Hour
	_ = p.DisableDeadInputs(httpClient, analyzerClient, sl1)
	assert.Contains(t, out.String(), `Disabling dead inputs of streams: progress "1 / 4 (25%)"`)
	assert.NotContains(t, out.String(), `progress "2 / 4 (50%)"`)
	assert.NotContains(t, out.String(), `progress "3 / 4 (75%)"`)
	assert.Contains(t, out.String(), `Disabling dead inputs of streams: progress "4 / 4 (100%)"`)
	assert.Contains(t, out.String(), `Disabling dead inputs of streams finished: checked "3", skipped "1", `+
		`disabled "2", took "`)

	// Log summary without progress if there are no inputs.
	out.Reset()
	_ = p.RemoveDeadInputs(httpClient, analyzerClient, []Stream{{}})
	assert.NotContains(t, out.String(), "progress")
	assert.Contains(t, out.String(), `Removing dead inputs from streams finished: checked "0", skipped "0", `+
		`removed "0", took "`)
}

func TestDisableDeadInputs(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.InputMaxConns = 100
	p.cfg.Streams.UseAnalyzer = false
	p.cfg.Streams.DeadInputsCheckBlacklist = []*regexp2.Regexp{
		regexp2.MustCompile(`ignore/1`, regexp2.RE2),
	}

	srv := newDeadInputsServers(t)
	httpClient := srv.Client(time.Second * 3)
	analyzerClient := analyzer.NewFake()

	// Check results.
	sl1 := []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{
				"http://no-such-host.test/1",
				"http://no-such-host.test/2",
				"http://streams.test/alive/1",
			},
			DisabledInputs: slicetest.WithSpare([]string{"http://dead/existing/1"}, "spare"),
		},
		{
			Name:   "Name 2",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://streams.test/alive/2", "http://no-such-host.test/3", "http://ignore/1"},
		},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.DisableDeadInputs(httpClient, analyzerClient, sl1)
	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, "spare", slicetest.Spare(sl1[0].DisabledInputs),
		"should not append to disabled inputs of `streams`")

	assert.Len(t, sl2, len(sl1))

	expected := []string{"http://streams.test/alive/1"}
	assert.Exactly(t, expected, sl2[0].Inputs)
	expected = []string{"http://dead/existing/1", "http://no-such-host.test/1", "http://no-such-host.test/2"}
	assert.Exactly(t, expected, sl2[0].DisabledInputs)

	expected = []string{"http://streams.test/alive/2", "http://ignore/1"}
	assert.Exactly(t, expected, sl2[1].Inputs)
	expected = []string{"http://no-such-host.test/3"}
	assert.Exactly(t, expected, sl2[1].DisabledInputs)

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.UseAnalyzer = false

	sl1 = []Stream{{
		ID:     "0",
		Name:   "Name 1",
		Groups: map[string]string{"Cat": "Grp"},
		Inputs: []string{"http://no-such-host.test/1"}},
	}

	_ = p.DisableDeadInputs(httpClient, analyzerClient, sl1)
	msg := `Disabling dead input of stream: ID "0", name "Name 1", group "Cat: Grp", ` +
		`input "http://no-such-host.test/1", reason "No such host"`
	assert.Contains(t, out.String(), msg)
}

func TestAddHashes(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.NameToInputHashMap = []cfg.HashAddRule{
		{By: regexp2.MustCompile(`Known name 1`, regexp2.RE2), Hash: "a"},
		{By: regexp2.MustCompile(`Known name 2`, regexp2.RE2), Hash: "b"},
	}
	p.cfg.Streams.GroupToInputHashMap = []cfg.HashAddRule{
		{By: regexp2.MustCompile(`Known group 1`, regexp2.RE2), Hash: "c"},
		{By: regexp2.MustCompile(`Known group 2`, regexp2.RE2), Hash: "d"},
	}
	p.cfg.Streams.InputToInputHashMap = []cfg.HashAddRule{
		{By: regexp2.MustCompile(`http://known/input/1`, regexp2.RE2), Hash: "e"},
		{By: regexp2.MustCompile(`http://known/input/2`, regexp2.RE2), Hash: "f"},
	}

	sl1 := []Stream{
		{ // Index 0. Known input 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/1#x", "http://other/input/1"},
		},
		{ // Index 1. Known name 1.
			Name:   "Known name 1",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1#a", "http://other/input/2#x"},
		},
		{ // Index 2. Known group 2.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://other/input/1#a&d", "http://other/input/2"},
		},
		{ // Index 3. Known inputs 2 and 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/2#x", "http://known/input/1"},
		},
		{ // Index 4. Known name 2.
			Name:   "Known name 2",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1"},
		},
		{ // Index 5. Known group 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://other/input/1#c", "http://other/input/2#x"},
		},
		{ // Index 6. No matches.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/2", "http://other/input/1#a"},
		},
		{ // Index 7. Matches by every parameter.
			Name:   "Known name 1",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://known/input/2#x", "http://other/input/1", "http://known/input/1"},
		},
		{ // Index 8. Matches by group 1 and input 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://known/input/1", "http://other/input/1"},
		},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.AddHashes(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := Stream{ // Index 0. Known input 1.
		Name:   "Other name",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://known/input/1#x&e", "http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[0], "inputs matching only by StreamInputToInputHashMap should get hashes only for"+
		"the exact inputs")

	expected = Stream{ // Index 1. Known name 1.
		Name:   "Known name 1",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://other/input/1#a", "http://other/input/2#x&a"},
	}
	assert.Exactly(t, expected, sl2[1], "should add hash to every matching input")

	expected = Stream{ // Index 2. Known group 2.
		Name:   "Other name",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		Inputs: []string{"http://other/input/1#a&d", "http://other/input/2#d"},
	}
	assert.Exactly(t, expected, sl2[2], "should add hash to every matching input")

	expected = Stream{ // Index 3. Known inputs 2 and 1.
		Name:   "Other name",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://known/input/2#x&f", "http://known/input/1#e"},
	}
	assert.Exactly(t, expected, sl2[3], "should add hash to every matching input")

	expected = Stream{ // Index 4. Known name 2.
		Name:   "Known name 2",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://other/input/1#b"},
	}
	assert.Exactly(t, expected, sl2[4], "should add hash to matching input")

	expected = Stream{ // Index 5. Known group 1.
		Name:   "Other name",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		Inputs: []string{"http://other/input/1#c", "http://other/input/2#x&c"},
	}
	assert.Exactly(t, expected, sl2[5], "should add hash to every matching input")

	// Index 6. No matches.
	assert.Exactly(t, sl1[6], sl2[6], "should not modify stream with no matches")

	expected = Stream{ // Index 7. Matches by every parameter.
		Name:   "Known name 1",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		Inputs: []string{"http://known/input/2#x&f&a&d", "http://other/input/1#a&d", "http://known/input/1#e&a&d"},
	}
	assert.Exactly(t, expected, sl2[7], "should add hash to every matching input by every parameter")

	expected = Stream{ // Index 8. Matches by group 1 and input 1.
		Name:   "Other name",
		Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		Inputs: []string{"http://known/input/1#e&c", "http://other/input/1#c"},
	}
	assert.Exactly(t, expected, sl2[8], "should add hash to every matching input by every parameter")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.NameToInputHashMap = []cfg.HashAddRule{
		{By: regexp2.MustCompile(`Name 1`, regexp2.RE2), Hash: "a"},
	}

	sl1 = []Stream{
		{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
	}

	_ = p.AddHashes(sl1)
	assert.Contains(t, out.String(), `Adding hash to input of stream: ID "0", name "Name 1", `+
		`group "Cat: Grp", hash "a", result "http://input/1#a"`)
}

func TestAllDisableAllButOneInputByRx(t *testing.T) {
	p := newDefProcessor()

	p.cfg.Streams.DisableAllButOneInputByRxList = []*regexp2.Regexp{
		regexp2.MustCompile(`[#&]no_sync(&|$)`, regexp2.RE2),
		regexp2.MustCompile(`[#&]i_might_stay(&|$)`, regexp2.RE2),
	}

	sl1 := []Stream{
		{
			Inputs: []string{
				"http://input/1#abc",
				"http://input/1#no_sync&abc",
				"http://input/1#abc&i_might_stay",
			},
			DisabledInputs: []string{"http://input/1#def"},
		},
		{
			Inputs:         []string{"http://input/1#abc"},
			DisabledInputs: []string{"http://input/1#no_sync&abc"},
		},
		{
			Inputs: []string{
				"http://input/1#abc&i_might_stay",
				"http://input/1#abc&i_might_stay",
				"http://input/1#abc",
			},
			DisabledInputs: []string{},
		},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.DisableAllButOneInputByRx(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []Stream{
		{
			Inputs:         []string{"http://input/1#no_sync&abc"},
			DisabledInputs: []string{"http://input/1#def", "http://input/1#abc", "http://input/1#abc&i_might_stay"},
		},
		{
			Inputs:         []string{"http://input/1#abc"},
			DisabledInputs: []string{"http://input/1#no_sync&abc"},
		},
		{
			Inputs:         []string{"http://input/1#abc&i_might_stay"},
			DisabledInputs: []string{"http://input/1#abc&i_might_stay", "http://input/1#abc"},
		},
	}
	assert.Exactly(t, expected, sl2)

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	p.cfg.Streams.DisableAllButOneInputByRxList = []*regexp2.Regexp{
		regexp2.MustCompile(`[#&]no_sync(&|$)`, regexp2.RE2),
	}

	sl1 = []Stream{
		{
			ID:     "0",
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://input/1#abc", "http://input/1#no_sync&abc"},
		},
	}

	_ = p.DisableAllButOneInputByRx(sl1)
	assert.Contains(t, out.String(), `Disabling other input per stream by regular expressions: ID "0", name "Name 1", `+
		`group "Cat: Grp", input "http://input/1#abc"`)
}

func TestAllRemoveDisabledInputs(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		{
			Inputs:         []string{"http://input/1", "http://input/2"},
			DisabledInputs: []string{"http://input/3", "http://input/4"},
		},
		{
			Inputs:         []string{"http://input/5"},
			DisabledInputs: []string{"http://input/6"},
		},
		{
			Inputs:         []string{},
			DisabledInputs: []string{},
		},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.RemoveDisabledInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := []Stream{
		{
			Inputs:         []string{"http://input/1", "http://input/2"},
			DisabledInputs: []string{},
		},
		{
			Inputs:         []string{"http://input/5"},
			DisabledInputs: []string{},
		},
		{
			Inputs:         []string{},
			DisabledInputs: []string{},
		},
	}
	assert.Exactly(t, expected, sl2)

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	sl1 = []Stream{
		{
			ID:             "0",
			Name:           "Name 1",
			Groups:         map[string]string{"Cat": "Grp"},
			DisabledInputs: []string{"http://input/1"},
		},
	}

	_ = p.RemoveDisabledInputs(sl1)
	assert.Contains(t, out.String(), `Removing disabled input: ID "0", name "Name 1", `+
		`group "Cat: Grp", input "http://input/1"`)
}

func TestSetKeepActive(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.NameToKeepActiveMap = []cfg.KeepActiveAddRule{
		{By: regexp2.MustCompile(`Known name 1`, regexp2.RE2), KeepActive: 0},
		{By: regexp2.MustCompile(`Known name 2`, regexp2.RE2), KeepActive: 1},
	}
	p.cfg.Streams.GroupToKeepActiveMap = []cfg.KeepActiveAddRule{
		{By: regexp2.MustCompile(`Known group 1`, regexp2.RE2), KeepActive: 2},
		{By: regexp2.MustCompile(`Known group 2`, regexp2.RE2), KeepActive: 3},
	}
	p.cfg.Streams.InputToKeepActiveMap = []cfg.KeepActiveAddRule{
		{By: regexp2.MustCompile(`http://known/input/1`, regexp2.RE2), KeepActive: 4},
		{By: regexp2.MustCompile(`http://known/input/2`, regexp2.RE2), KeepActive: 5},
	}

	sl1 := []Stream{
		{ // Index 0. Known input 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/1", "http://other/input/1"},
		},
		{ // Index 1. Known name 1.
			Name:   "Known name 1",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1", "http://other/input/2"},
		},
		{ // Index 2. Known group 2.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://other/input/1", "http://other/input/2"},
		},
		{ // Index 3. Known inputs 2 and 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/2", "http://known/input/1"},
		},
		{ // Index 4. Known name 2 and group 1.
			Name:   "Known name 2",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://other/input/1"},
		},
		{ // Index 5. Known group 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://other/input/1", "http://other/input/2"},
		},
		{ // Index 6. No matches.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/2", "http://other/input/1"},
		},
		{ // Index 7. Matches by every parameter.
			Name:   "Known name 1",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://other/input/1", "http://known/input/2"},
		},
		{ // Index 8. Matches by group 1 and input 1.
			Name:   "Other name",
			Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://known/input/1", "http://other/input/1"},
		},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.SetKeepActive(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	expected := Stream{ // Index 0. Known input 1.
		Name:           "Other name",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
		HTTPKeepActive: "4",
		Inputs:         []string{"http://known/input/1", "http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[0])

	expected = Stream{ // Index 1. Known name 1.
		Name:           "Known name 1",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
		HTTPKeepActive: "0",
		Inputs:         []string{"http://other/input/1", "http://other/input/2"},
	}
	assert.Exactly(t, expected, sl2[1])

	expected = Stream{ // Index 2. Known group 2.
		Name:           "Other name",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		HTTPKeepActive: "3",
		Inputs:         []string{"http://other/input/1", "http://other/input/2"},
	}
	assert.Exactly(t, expected, sl2[2])

	expected = Stream{ // Index 3. Known inputs 2 and 1.
		Name:           "Other name",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Other group"},
		HTTPKeepActive: "4",
		Inputs:         []string{"http://known/input/2", "http://known/input/1"},
	}
	assert.Exactly(t, expected, sl2[3], "should set HTTPKeepActive by first found rule by inputs")

	expected = Stream{ // Index 4. Known name 2 and group 1.
		Name:           "Known name 2",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		HTTPKeepActive: "1",
		Inputs:         []string{"http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[4], "should set HTTPKeepActive by name")

	expected = Stream{ // Index 5. Known group 1.
		Name:           "Other name",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		HTTPKeepActive: "2",
		Inputs:         []string{"http://other/input/1", "http://other/input/2"},
	}
	assert.Exactly(t, expected, sl2[5])

	// Index 6. No matches.
	assert.Exactly(t, sl1[6], sl2[6], "should not modify stream with no matches")

	expected = Stream{ // Index 7. Matches by every parameter.
		Name:           "Known name 1",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		HTTPKeepActive: "5",
		Inputs:         []string{"http://other/input/1", "http://known/input/2"},
	}
	assert.Exactly(t, expected, sl2[7], "should set HTTPKeepActive by first found rule by inputs")

	expected = Stream{ // Index 8. Matches by group 1 and input 1.
		Name:           "Other name",
		Groups:         map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		HTTPKeepActive: "4",
		Inputs:         []string{"http://known/input/1", "http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[8], "should set HTTPKeepActive by first found rule by inputs")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.NameToKeepActiveMap = []cfg.KeepActiveAddRule{
		{By: regexp2.MustCompile(`Name 1`, regexp2.RE2), KeepActive: 1},
	}

	sl1 = []Stream{{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}}}

	_ = p.SetKeepActive(sl1)
	assert.Contains(t, out.String(), `Setting keep active on stream: ID "0", name "Name 1", `+
		`group "Cat: Grp", keep active "1"`)
}

func TestRemoveWithoutInputs(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		{Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Group"}},
		{Enabled: true, Name: "Name"},
		{Enabled: true, Inputs: []string{"http://input/1", "http://input/2"}},
		{Enabled: false, Name: p.cfg.Streams.DisabledPrefix + "Name"},
		{Inputs: []string{"http://input"}},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.RemoveWithoutInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	expected := []Stream{
		{Groups: map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Group"}, Remove: true},
		{Enabled: true, Name: "Name", Remove: true},
		{Enabled: true, Inputs: []string{"http://input/1", "http://input/2"}},
		{Enabled: false, Name: p.cfg.Streams.DisabledPrefix + "Name", Remove: true},
		{Inputs: []string{"http://input"}},
	}

	assert.Exactly(t, expected, sl2, "should remove streams without inputs")

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	sl1 = []Stream{
		{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}},
		{ID: "1", Name: "Name 2", Groups: map[string]string{"Cat": "Grp"}, Remove: true},
	}

	_ = p.RemoveWithoutInputs(sl1)
	assert.Contains(t, out.String(), `Removing stream without inputs: ID "0", name "Name 1", group "Cat: Grp"`)
	assert.NotContains(t, out.String(), `Removing stream without inputs: ID "1", name "Name 2", group "Cat: Grp"`)
}

func TestDisableWithoutInputs(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		/* 0 */ {MarkDisabled: false, Enabled: true, Name: "Name", Inputs: []string{"http://input"}},
		/* 1 */ {MarkDisabled: false, Enabled: true, Name: "Name", Inputs: []string{}},
		/* 2 */ {MarkDisabled: false, Enabled: false, Name: "Name", Inputs: []string{"http://input"}},
		/* 3 */ {MarkDisabled: false, Enabled: false, Name: "Name"},

		/* 4 */ {MarkDisabled: true, Enabled: true, Name: "Name", Inputs: []string{"http://input"}},
		/* 5 */ {MarkDisabled: true, Enabled: true, Name: "Name", Inputs: []string{}},
		/* 6 */ {MarkDisabled: true, Enabled: false, Name: "Name", Inputs: []string{"http://input"}},
		/* 7 */ {MarkDisabled: true, Enabled: false, Name: "Name"},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.DisableWithoutInputs(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	assert.Exactly(t, sl1[0], sl2[0], "should not modify the stream with inputs")

	expected := Stream{MarkDisabled: true, Enabled: false, Name: "Name", Inputs: []string{}}
	assert.Exactly(t, expected, sl2[1], "should disable stream without inputs and set MarkDisabled to true")

	assert.Exactly(t, sl1[2], sl2[2], "should not modify disabled stream or stream with inputs")

	assert.Exactly(t, sl1[3], sl2[3])

	assert.Exactly(t, sl1[4], sl2[4], "should not modify the stream with inputs")

	expected = Stream{MarkDisabled: true, Enabled: false, Name: "Name", Inputs: []string{}}
	assert.Exactly(t, expected, sl2[5], "should disable stream without inputs")

	assert.Exactly(t, sl1[6], sl2[6], "should not modify disabled stream or stream with inputs")

	assert.Exactly(t, sl1[7], sl2[7])

	// Test log output.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)

	sl1 = []Stream{{ID: "0", Enabled: true, Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}}}

	_ = p.DisableWithoutInputs(sl1)
	assert.Contains(t, out.String(), `Disabling stream without inputs: ID "0", name "Name 1", group "Cat: Grp"`)
}

func TestAddNamePrefixes(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		/* 0 */ {MarkAdded: false, MarkDisabled: false, Name: "Name_1"},
		/* 1 */ {MarkAdded: true, MarkDisabled: false, Name: "Name_2"},
		/* 2 */ {MarkAdded: false, MarkDisabled: true, Name: "Name_3"},
		/* 3 */ {MarkAdded: true, MarkDisabled: true, Name: "Name_4"},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := p.AddNamePrefixes(sl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.Len(t, sl2, len(sl1))

	assert.Exactly(t, sl1[0], sl2[0], "should not modify the stream with MarkAdded and MarkDisabled set to false")

	expected := Stream{MarkAdded: true, MarkDisabled: false, Name: p.cfg.Streams.AddedPrefix + "Name_2"}
	assert.Exactly(t, expected, sl2[1])

	expected = Stream{MarkAdded: false, MarkDisabled: true, Name: p.cfg.Streams.DisabledPrefix + "Name_3"}
	assert.Exactly(t, expected, sl2[2])

	expected = Stream{MarkAdded: true, MarkDisabled: true,
		Name: p.cfg.Streams.DisabledPrefix + p.cfg.Streams.AddedPrefix + "Name_4"}
	assert.Exactly(t, expected, sl2[3])

	// Check if logs are not printed if prefixes are empty.
	var out syncBuffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.AddedPrefix = ""
	p.cfg.Streams.DisabledPrefix = ""

	sl1 = []Stream{{MarkAdded: true, Name: "Name_1"}}

	_ = p.AddNamePrefixes(sl1)
	assert.NotContains(t, out.String(), "Name_1")

	// Check if logs are not printed if prefixes are empty.
	out.Reset()
	p = newDefProcessorWithWriter(&out)

	sl1 = []Stream{{ID: "0", MarkAdded: true, Name: "Name_1", Groups: map[string]string{"Cat": "Grp"}}}

	_ = p.AddNamePrefixes(sl1)
	assert.Contains(t, out.String(), fmt.Sprintf(`Adding name prefix to stream: ID "0", old name "Name_1", `+
		`new name "%vName_1", group "Cat: Grp"`, p.cfg.Streams.AddedPrefix))
}

func TestChangedStreams(t *testing.T) {
	p := newDefProcessor()

	sl1 := []Stream{
		{Name: "Stream 1", ID: "0001", Enabled: true, Inputs: []string{"A", "B"}},
		{Name: "Stream 2", ID: "0002", Enabled: false, Inputs: []string{"C", "D"}},
		{Name: "Stream 3", ID: "0003", Enabled: false, Inputs: []string{"E", "F"}},
		{Name: "Stream 4", ID: "0004", Enabled: true, Inputs: []string{"G", "H"}, Groups: map[string]string{"A": "B"}},
		{Name: "Stream 5", ID: "0005"},
		{Name: "Stream 6", ID: "0006", Enabled: true, Inputs: []string{"I", "J"}, MarkAdded: true, MarkDisabled: false},
	}
	sl1Original := copier.MustDeep(sl1)

	sl2 := []Stream{
		// No changes.
		{Name: "Stream 1", ID: "0001", Enabled: true, Inputs: []string{"A", "B"}},
		// Changed inputs.
		{Name: "Stream 2", ID: "0002", Enabled: false, Inputs: []string{"C2", "D"}},
		// No changes.
		{Name: "Stream 3", ID: "0003", Enabled: false, Inputs: []string{"E", "F"}},
		// Changed groups.
		{Name: "Stream 4", ID: "0004", Enabled: true, Inputs: []string{"G", "H"}, Groups: map[string]string{"C": "D"}},
		// New.
		{Name: "Stream 7", ID: "0007", Enabled: true, Inputs: []string{"K", "L"}, Groups: map[string]string{"E": "F"}},
		// Changed name.
		{Name: "Stream 5*", ID: "0005"},
		// Changed MarkAdded / MarkDisabled (no changes).
		{Name: "Stream 6", ID: "0006", Enabled: true, Inputs: []string{"I", "J"}, MarkAdded: false, MarkDisabled: true},
		// New.
		{Name: "Stream 8", ID: "0008", Enabled: false, DisabledInputs: []string{"A", "B"}},
	}
	sl2Original := copier.MustDeep(sl2)

	actual := p.ChangedStreams(sl1, sl2)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, sl2Original, sl2)

	expected := []Stream{
		{Name: "Stream 2", ID: "0002", Enabled: false, Inputs: []string{"C2", "D"}},
		{Name: "Stream 4", ID: "0004", Enabled: true, Inputs: []string{"G", "H"}, Groups: map[string]string{"C": "D"}},
		{Name: "Stream 7", ID: "0007", Enabled: true, Inputs: []string{"K", "L"}, Groups: map[string]string{"E": "F"}},
		{Name: "Stream 5*", ID: "0005"},
		{Name: "Stream 8", ID: "0008", Enabled: false, DisabledInputs: []string{"A", "B"}},
	}
	assert.Exactly(t, expected, actual)
}

func TestGetInputsAmount(t *testing.T) {
	sl1 := []Stream{
		{Inputs: []string{"http://input/1"}},
		{Inputs: []string{"http://input/1", "http://input/1"}},
		{Inputs: []string{"http://input/1", "http://input/2"}},
		{Inputs: []string{"http://input/1"}},
		{},
	}

	assert.Exactly(t, 6, getInputsAmount(sl1))
}
