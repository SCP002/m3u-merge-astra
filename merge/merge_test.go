package merge

import (
	"bytes"
	"m3u-merge-astra/astra"
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/m3u"
	"m3u-merge-astra/util/copier"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenameStreams(t *testing.T) {
	p := newDefProcessor()

	sl1 := []astra.Stream{
		{Name: "Known name 2"}, {Name: "Known name"}, {Name: "Other name A"},
	}
	sl1Original := copier.MustDeep(sl1)
	cl1 := []m3u.Channel{
		{Name: "Other_Name_B"}, {Name: "Known_Name_2"}, {Name: "Known_Name"},
	}
	cl1Original := copier.MustDeep(cl1)

	sl2 := p.RenameStreams(sl1, cl1)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, len(sl1))

	expected := astra.Stream{Name: "Known_Name_2"}
	assert.Exactly(t, expected, sl2[0])

	expected = astra.Stream{Name: "Known_Name"}
	assert.Exactly(t, expected, sl2[1])

	expected = astra.Stream{Name: "Other name A"}
	assert.Exactly(t, expected, sl2[2], "should not rename stream if no channel counterpart name found")

	// Test log output.
	var out bytes.Buffer
	p = newDefProcessorWithWriter(&out)

	sl1 = []astra.Stream{{ID: "0", Name: "Known name", Groups: map[string]string{"Cat": "Grp"}}}

	cl1 = []m3u.Channel{{Name: "Known_Name"}}

	_ = p.RenameStreams(sl1, cl1)
	assert.Contains(t, out.String(), `Renaming stream: ID "0", old name "Known name", `+
		`new name "Known_Name", group "Cat: Grp"`)
}

func TestUpdateInputs(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/2"), To: *regexp.MustCompile("known/input/2")},
		{From: *regexp.MustCompile("known/input/1"), To: *regexp.MustCompile("known/input/1")},
	}

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{
			"http://other/input", "http://known/input/2#a", "http://known/input/1", "http://known/input/1#b",
		}},
		{Name: "Known name 2", Inputs: []string{"http://known/input/2#b"}},
		{Name: "Other name A", Inputs: []string{"http://known/input/1#a"}},
	}
	sl1Original := copier.MustDeep(sl1)
	cl1 := []m3u.Channel{
		{Name: "Other_Name_B", URL: "http://new/known/input/1#a"},
		{Name: "Known_Name", URL: "http://new/known/input/2#b"},
		{Name: "Known_Name_2", URL: "http://other/url"},
		{Name: "Known-Name", URL: "http://new/known/input/1"},
		{Name: "KnownName", URL: "http://_other_/input"},
	}
	cl1Original := copier.MustDeep(cl1)

	p.cfg.Streams.KeepInputHash = false
	sl2 := p.UpdateInputs(sl1, cl1)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, len(sl1))

	expected := []string{
		"http://other/input", "http://new/known/input/2#b", "http://new/known/input/1", "http://known/input/1#b",
	}
	assert.Exactly(t, expected, sl2[0].Inputs)

	assert.Exactly(t, sl1[1], sl2[1], "should not update to input to channel URL not in cfg.Streams.InputUpdateMap")

	assert.Exactly(t, sl1[2], sl2[2], "only streams with known names should be updated")

	p.cfg.Streams.KeepInputHash = true
	sl2 = p.UpdateInputs(sl1, cl1)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, len(sl1))

	expected = []string{
		"http://other/input", "http://new/known/input/2#b&a", "http://new/known/input/1", "http://known/input/1#b",
	}
	assert.Exactly(t, expected, sl2[0].Inputs)

	assert.Exactly(t, sl1[1], sl2[1], "should not update to input to channel URL not in cfg.Streams.InputUpdateMap")

	assert.Exactly(t, sl1[2], sl2[2], "only streams with known names should be updated")

	// Test Streams.EnableOnInputUpdate.
	p.cfg.Streams.EnableOnInputUpdate = false
	sl1 = []astra.Stream{
		{Enabled: false, MarkDisabled: true, Name: "Known name", Inputs: []string{"http://known/input/1"}},
		{Enabled: false, MarkDisabled: true, Name: "Known name", Inputs: []string{"http://other/input/1"}},
		{Enabled: false, MarkDisabled: true, Name: "Other name", Inputs: []string{"http://known/input/1"}},
	}
	sl1Original = copier.MustDeep(sl1)
	cl1 = []m3u.Channel{
		{Name: "Known name", URL: "http://new/known/input/1"},
	}

	sl2 = p.UpdateInputs(sl1, cl1)

	assert.False(t, sl2[0].Enabled, "stream should stay disabled as EnableOnInputUpdate = false")
	assert.True(t, sl2[0].MarkDisabled, "MarkDisabled should stay disabled as EnableOnInputUpdate = false")
	assert.False(t, sl2[1].Enabled, "stream should stay disabled as EnableOnInputUpdate = false")
	assert.True(t, sl2[1].MarkDisabled, "MarkDisabled should stay disabled as EnableOnInputUpdate = false")
	assert.False(t, sl2[2].Enabled, "stream should stay disabled as EnableOnInputUpdate = false")
	assert.True(t, sl2[2].MarkDisabled, "MarkDisabled should stay disabled as EnableOnInputUpdate = false")

	p.cfg.Streams.EnableOnInputUpdate = true

	sl2 = p.UpdateInputs(sl1, cl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.True(t, sl2[0].Enabled, "stream should become enabled as EnableOnInputUpdate = true and it was updated")
	assert.False(t, sl2[0].MarkDisabled,
		"MarkDisabled should become false as EnableOnInputUpdate = true and it was updated")
	assert.False(t, sl2[1].Enabled, "stream should stay disabled as it's input is not in cfg.Streams.InputUpdateMap")
	assert.True(t, sl2[1].MarkDisabled,
		"MarkDisabled should stay true as it's input is not in cfg.Streams.InputUpdateMap")
	assert.False(t, sl2[2].Enabled, "stream should stay disabled as it's name is not in channel list")
	assert.True(t, sl2[2].MarkDisabled,
		"MarkDisabled should stay true as it's name is not in channel list")

	// Test log output.
	var out bytes.Buffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.EnableOnInputUpdate = false
	p.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/1"), To: *regexp.MustCompile("known/input/1")},
	}

	sl1 = []astra.Stream{{ID: "0", Enabled: false, Name: "Known name", Inputs: []string{"http://known/input/1"}}}

	cl1 = []m3u.Channel{{Name: "Known_Name", URL: "http://new/known/input/1"}}

	_ = p.UpdateInputs(sl1, cl1)
	assert.Contains(t, out.String(), `Updating input of stream: ID "0", name "Known name", `+
		`old URL "http://known/input/1", new URL "http://new/known/input/1", note "Stream is disabled"`)

	out.Reset()
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.EnableOnInputUpdate = true
	p.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/1"), To: *regexp.MustCompile("known/input/1")},
	}

	sl1 = []astra.Stream{{ID: "0", Enabled: false, Name: "Known name", Inputs: []string{"http://known/input/1"}}}

	cl1 = []m3u.Channel{{Name: "Known_Name", URL: "http://new/known/input/1"}}

	_ = p.UpdateInputs(sl1, cl1)
	assert.Contains(t, out.String(), `Enabling the stream (updating inputs of streams, enable_on_input_update is on): `+
		`ID "0", name "Known name"`)
}

func TestRemoveInputsByUpdateMap(t *testing.T) {
	p := newDefProcessor()
	p.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/1")},
		{From: *regexp.MustCompile("known/input/2")},
	}

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://other/input", "http://known/input/2", "http://known/input/1#a"}},
		{Name: "Known name 2", Inputs: []string{"http://known/input/2#a"}},
		{
			Name:   "Other name A",
			Inputs: []string{"http://known/input/1#a", "http://known/input/2", "http://other/input"},
		},
	}
	sl1Original := copier.MustDeep(sl1)
	cl1 := []m3u.Channel{
		{Name: "Other_Name_B", URL: "http://other/url"},
		{Name: "Known_Name", URL: "http://known/input/1#b"},
		{Name: "Known_Name_2", URL: "http://other/url"},
		{Name: "Known_Name_2", URL: "http://known/input/2"},
	}
	cl1Original := copier.MustDeep(cl1)
	sl2 := p.RemoveInputsByUpdateMap(sl1, cl1)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, len(sl1))

	expected := astra.Stream{
		Name:           "Known name",
		Inputs:         []string{"http://other/input", "http://known/input/1#a"},
		DisabledInputs: make([]string, 0),
	}
	assert.Exactly(t, expected, sl2[0], "unknown inputs and inputs which only differ by hash should stay")

	expected = astra.Stream{Name: "Known name 2", Inputs: []string{"http://known/input/2#a"}}
	assert.Exactly(t, expected, sl2[1], "known inputs which only differ by hash should stay")

	expected = astra.Stream{
		Name:           "Other name A",
		Inputs:         []string{"http://other/input"},
		DisabledInputs: make([]string, 0),
	}
	assert.Exactly(t, expected, sl2[2], "known inputs not found in channels should be removed, unknown should stay")

	// Test log output.
	var out bytes.Buffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{{From: *regexp.MustCompile("known/input/1")}}

	sl1 = []astra.Stream{
		{
			ID:     "0",
			Name:   "Known name",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://known/input/1"},
		},
	}

	cl1 = []m3u.Channel{{Name: "Known_Name", URL: "http://other/url"}}

	_ = p.RemoveInputsByUpdateMap(sl1, cl1)
	assert.Contains(t, out.String(), `Removing absent input from stream according the update map: ID "0", `+
		`name "Known name", group "Cat: Grp", input "http://known/input/1"`)
}

func TestAddNewInputs(t *testing.T) {
	p := newDefProcessor()

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://input/1#a", "http://input/2"}},
		{Name: "Known name 2", Inputs: []string{"http://input/1"}},
		{Name: "Known name 3", Inputs: []string{"http://input/1"}},
		{Name: "Other name A", Inputs: []string{"http://input/1#b"}},
	}
	sl1Original := copier.MustDeep(sl1)
	cl1 := []m3u.Channel{
		{Name: "Known_Name_2", URL: "http://input/1"},
		{Name: "Known_Name", URL: "http://input/2#a"},
		{Name: "Known_Name_3", URL: "http://input/2"},
		{Name: "Other_Name_B", URL: "http://input/1#c"},
		{Name: "Known-Name-3", URL: "http://input/3"},
	}
	cl1Original := copier.MustDeep(cl1)

	p.cfg.Streams.HashCheckOnAddNewInputs = true
	sl2 := p.AddNewInputs(sl1, cl1)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, len(sl1))

	expected := astra.Stream{
		Name:   "Known name",
		Inputs: []string{"http://input/2#a", "http://input/1#a", "http://input/2"},
	}
	assert.Exactly(t, expected, sl2[0], "should add input which only differ by hash as HashCheckOnAddNewStreamInputs"+
		" = true")

	expected = astra.Stream{Name: "Known name 2", Inputs: []string{"http://input/1"}}
	assert.Exactly(t, expected, sl2[1], "should add only unknown inputs")

	expected = astra.Stream{
		Name:   "Known name 3",
		Inputs: []string{"http://input/3", "http://input/2", "http://input/1"},
	}
	assert.Exactly(t, expected, sl2[2])

	assert.Exactly(t, sl1[3], sl2[3], "should not change streams with unknown names")

	p.cfg.Streams.HashCheckOnAddNewInputs = false
	sl2 = p.AddNewInputs(sl1, cl1)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, len(sl1))

	assert.Exactly(t, sl1[0], sl2[0], "should not add input which only differ by hash as HashCheckOnAddNewStreamInputs"+
		" = false")

	assert.Exactly(t, sl1[1], sl2[1], "should add only unknown inputs")

	expected = astra.Stream{
		Name:   "Known name 3",
		Inputs: []string{"http://input/3", "http://input/2", "http://input/1"},
	}
	assert.Exactly(t, expected, sl2[2])

	assert.Exactly(t, sl1[3], sl2[3], "should not change streams with unknown names")

	// Test Streams.EnableOnInputUpdate.
	p.cfg.Streams.EnableOnInputUpdate = false
	sl1 = []astra.Stream{
		{Enabled: false, MarkDisabled: true, Name: "Known name", Inputs: []string{"http://input/1"}},
	}
	sl1Original = copier.MustDeep(sl1)
	cl1 = []m3u.Channel{
		{Name: "Known name", URL: "http://input/2"},
	}

	sl2 = p.AddNewInputs(sl1, cl1)

	assert.False(t, sl2[0].Enabled, "stream should stay disabled as EnableOnInputUpdate = false")
	assert.True(t, sl2[0].MarkDisabled, "MarkDisabled should stay true as EnableOnInputUpdate = false")

	p.cfg.Streams.EnableOnInputUpdate = true

	sl2 = p.AddNewInputs(sl1, cl1)
	assert.Exactly(t, sl1Original, sl1)

	assert.True(t, sl2[0].Enabled, "stream should become enabled as EnableOnInputUpdate = true")
	assert.False(t, sl2[0].MarkDisabled, "MarkDisabled should become false as EnableOnInputUpdate = true")

	sl1 = []astra.Stream{
		{Enabled: false, MarkDisabled: true, Name: "Other name", Inputs: []string{"http://input/1"}},
	}

	sl2 = p.AddNewInputs(sl1, cl1)

	assert.Exactly(t, sl1, sl2, "should stay the same because it was not updated")

	// Test log output.
	var out bytes.Buffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.EnableOnInputUpdate = false

	sl1 = []astra.Stream{
		{ID: "0", Enabled: false, Name: "Known name", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{}},
	}

	cl1 = []m3u.Channel{{Name: "Known_Name", URL: "http://url/1"}}

	_ = p.AddNewInputs(sl1, cl1)
	assert.Contains(t, out.String(), `Adding new input to stream: ID "0", name "Known name", group "Cat: Grp", `+
		`URL "http://url/1", note "Stream is disabled"`)

	out.Reset()
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.EnableOnInputUpdate = true

	sl1 = []astra.Stream{
		{ID: "0", Enabled: false, Name: "Known name", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{}},
	}

	cl1 = []m3u.Channel{{Name: "Known_Name", URL: "http://url/1"}}

	_ = p.AddNewInputs(sl1, cl1)
	assert.Contains(t, out.String(), `Enabling the stream (adding new inputs to streams, `+
		`enable_on_input_update is on): ID "0", name "Known name"`)
}

func TestAddNewStreams(t *testing.T) {
	p := newDefProcessor()

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://some/url"}},
		{Name: "Other name A", Inputs: []string{"http://some/url/2"}},
	}
	sl1Original := copier.MustDeep(sl1)
	cl1 := []m3u.Channel{
		{Name: "Other name B", Group: "Group", URL: "http://some/url/2"},
		{Name: "Known name", URL: "http://some/url"},
		{Name: "Other name B", Group: "Group", URL: "http://some/url/3"},
	}
	cl1Original := copier.MustDeep(cl1)

	p.cfg.Streams.AddNewWithKnownInputs = true
	p.cfg.Streams.AddGroupsToNew = false
	sl2, gid := p.AddNewStreams(sl1, cl1, 0)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, 3)

	assert.Exactly(t, sl1[0], sl2[0])

	assert.Exactly(t, sl1[1], sl2[1])

	expected := astra.Stream{
		DisabledInputs: make([]string, 0),
		Enabled:        p.cfg.Streams.MakeNewEnabled,
		HTTPKeepActive: strconv.Itoa(p.cfg.Streams.NewKeepActive),
		ID:             "a001",
		Inputs:         []string{"http://some/url/2"},
		Name:           "Other name B",
		Type:           string(p.cfg.Streams.NewType),
		MarkAdded:      true,
	}
	assert.Exactly(t, expected, sl2[2])
	assert.Exactly(t, 466561, gid)

	p.cfg.Streams.AddNewWithKnownInputs = true
	p.cfg.Streams.AddGroupsToNew = true
	sl2, gid = p.AddNewStreams(sl1, cl1, gid)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Len(t, sl2, 3)

	assert.Exactly(t, sl1[0], sl2[0])

	assert.Exactly(t, sl1[1], sl2[1])

	expected.ID = "a002"
	expected.Groups = map[string]string{p.cfg.Streams.GroupsCategoryForNew: "Group"}
	assert.Exactly(t, expected, sl2[2])
	assert.Exactly(t, 466562, gid)

	sl1 = []astra.Stream{{Name: "Other name", Inputs: []string{"http://some/url"}}}
	cl1 = []m3u.Channel{
		{Name: "Other name", URL: "http://some/url"},
		{Name: "Other name", URL: "http://some/url/2"},
	}
	sl2, gid = p.AddNewStreams(sl1, cl1, 466562)

	assert.Exactly(t, sl1, sl2, "should not change as M3U channels does not contain any new name")
	assert.Exactly(t, 466562, gid, "should not change as no IDs generated")

	sl1 = []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://some/url#a"}},
		{Name: "Other name A", Inputs: []string{"http://known/url#b"}},
	}
	sl1Original = copier.MustDeep(sl1)
	cl1 = []m3u.Channel{
		{Name: "Other name B", URL: "http://known/url#c"},
		{Name: "Known name", URL: "http://some/url/2#a"},
	}
	cl1Original = copier.MustDeep(cl1)

	p.cfg.Streams.AddNewWithKnownInputs = false
	sl2, _ = p.AddNewStreams(sl1, cl1, 0)

	assert.Exactly(t, sl1Original, sl1)
	assert.Exactly(t, cl1Original, cl1)

	assert.Exactly(t, sl1, sl2, "should not change as AddNewStreamsWithKnownInputs = false and hash difference should"+
		"be ignored")

	// Test log output.
	var out bytes.Buffer
	p = newDefProcessorWithWriter(&out)
	p.cfg.Streams.AddGroupsToNew = true

	sl1 = []astra.Stream{}

	cl1 = []m3u.Channel{{Name: "Name 1", Group: "Grp", URL: "http://url/1"}}

	_, _ = p.AddNewStreams(sl1, cl1, 0)
	assert.Contains(t, out.String(), `Adding new stream: ID "a001", name "Name 1", group "All: Grp", `+
		`input "http://url/1"`)
}

func TestGenerateUID(t *testing.T) {
	uid, gid := generateUID(nil, 0)
	assert.Exactly(t, "a001", uid, "should start from default counter if not set")
	assert.Exactly(t, 466561, gid)

	uid, gid = generateUID(nil, gid)
	assert.Exactly(t, "a002", uid)
	assert.Exactly(t, 466562, gid)

	sl := []astra.Stream{{ID: "a003"}, {ID: "a004"}, {ID: "a006"}}
	uid, gid = generateUID(sl, gid)
	assert.Exactly(t, "a005", uid, "should skip IDs used by streams")
	assert.Exactly(t, 466565, gid)

	uid, gid = generateUID(nil, 1679615) // "zzzz".
	assert.Exactly(t, "10000", uid)
	assert.Exactly(t, 1679616, gid)

	uid, gid = generateUID(nil, 35)
	assert.Exactly(t, "10", uid)
	assert.Exactly(t, 36, gid)
}
