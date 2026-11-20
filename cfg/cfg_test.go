package cfg

import (
	"m3u-merge-astra/util/copier"
	"m3u-merge-astra/util/file"
	"m3u-merge-astra/util/logger"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dlclark/regexp2"
	"github.com/goccy/go-yaml"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestSimplifyAliases(t *testing.T) {
	cfg := General{
		NameAliasList: [][]string{
			{"Name 1", "Name 1 var 2", "Name 1 var 3"},
			{"Name_2", "Name_2_var_2"},
		},
	}
	cfgOriginal := copier.MustDeep(cfg)

	actual := cfg.SimplifyAliases()

	assert.Exactly(t, cfgOriginal, cfg)

	expected := [][]string{
		{"name1", "name1var2", "name1var3"},
		{"name2", "name2var2"},
	}
	assert.Exactly(t, expected, actual)
}

func TestBadRegexpError(t *testing.T) {
	err := error(BadRegexpError{Reason: "Bad", Regexp: regexp2.MustCompile(`.*`, regexp2.RE2)})
	expected := "Bad; Regular expression: .*"
	assert.Exactly(t, expected, err.Error())
}

func TestInitDefault(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	path := filepath.Join(t.TempDir(), "m3u-merge-astra-init-test.yaml")

	// Test creation of the default config.
	actual, isNewCfg, err := Init(log, path)

	assert.Exactly(t, Root{}, actual)
	assert.True(t, isNewCfg)
	assert.NoError(t, err)

	// Test reading of the default config.
	actual, isNewCfg, err = Init(log, path)

	expected := NewDefCfg()

	assert.Exactly(t, expected, actual)
	assert.False(t, isNewCfg)
	assert.NoError(t, err)
}

func TestInitValidateCaptureGroups(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	path := filepath.Join(t.TempDir(), "m3u-merge-astra-init-test.yaml")

	// Test reading config with 'remove_duplicated_inputs_by_rx_list'.
	err := file.Copy("init-validate-capture-groups-test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	_, isNewCfg, err := Init(log, path)

	expectedErr := BadRegexpError{
		Regexp: regexp2.MustCompile(`rx_without_capture_group`, regexp2.RE2),
		Reason: "Expecting at least one capture group",
	}

	assert.False(t, isNewCfg)
	assert.Exactly(t, expectedErr, errors.UnwrapAll(err))
}

func TestInitSimplifyAliases(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	path := filepath.Join(t.TempDir(), "m3u-merge-astra-init-test.yaml")

	// Test reading config with name aliases and simplification of them.
	err := file.Copy("init-simplify-aliases-test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	actual, isNewCfg, err := Init(log, path)

	expected := [][]string{
		{"sample", "sampletv", "sampletelevisionchannel"},
		{"discoveryid", "discoveryinvestigation"},
	}

	assert.Exactly(t, expected, actual.General.SimpleNameAliasList)
	assert.False(t, isNewCfg)
	assert.NoError(t, err)
}

func TestAddNewOption(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	path := "$.streams.new_option"
	defComment := yaml.HeadComment("", " New option.")
	defCommentMap := yaml.CommentMap{path: []*yaml.Comment{defComment}}

	// Missing option.
	var opt *int
	commentMap := yaml.CommentMap{}
	assert.True(t, addNewOption(log, &opt, 5, path, commentMap, defCommentMap), "should add missing option")
	assert.Exactly(t, 5, *opt, "should set default value")
	assert.Exactly(t, defCommentMap, commentMap, "should add default comment")

	// Present option, even with zero value.
	opt = new(0)
	commentMap = yaml.CommentMap{}
	assert.False(t, addNewOption(log, &opt, 5, path, commentMap, defCommentMap), "should not add present option")
	assert.Exactly(t, 0, *opt, "should keep value")
	assert.Empty(t, commentMap, "should not add comment")
}

func TestInitRegexpWithLookaround(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	path := filepath.Join(t.TempDir(), "m3u-merge-astra-init-test.yaml")
	cfgBytes := []byte("streams:\n  input_blacklist:\n  - '^http://(?!keep\\.)'\n  - '(?<=#)drop$'\n")
	err := os.WriteFile(path, cfgBytes, 0644)
	assert.NoError(t, err)

	root, isNewCfg, err := Init(log, path)
	assert.NoError(t, err)
	assert.False(t, isNewCfg)

	negativeLookahead := root.Streams.InputBlacklist[0]
	assert.Exactly(t, time.Second, negativeLookahead.MatchTimeout, "should set match timeout")
	match, err := negativeLookahead.MatchString("http://other.tv/1")
	assert.NoError(t, err)
	assert.True(t, match, "should support negative lookahead")
	match, err = negativeLookahead.MatchString("http://keep.tv/1")
	assert.NoError(t, err)
	assert.False(t, match, "should support negative lookahead")

	lookbehind := root.Streams.InputBlacklist[1]
	match, err = lookbehind.MatchString("http://url/1#drop")
	assert.NoError(t, err)
	assert.True(t, match, "should support lookbehind")
}

func TestEncodeRegexp(t *testing.T) {
	// Patterns with characters special to YAML.
	patterns := []string{`^.*:\/\/([^#?/]*)`, `[#&]no_sync(&|$)`, `x #y`, `it's`, `a: b`, `&anchor`, `!bang`, `%pct`,
		`- dash`, `'quoted'`, `"dq"`, `{x}`, `[ab]`, `^http://(?!keep\.)`}
	var root Root
	for _, pattern := range patterns {
		root.Streams.InputBlacklist = append(root.Streams.InputBlacklist, regexp2.MustCompile(pattern, regexp2.RE2))
	}

	bytes, err := encode(root, yaml.CommentMap{})
	assert.NoError(t, err)

	var decoded Root
	err = yaml.Unmarshal(bytes, &decoded)
	assert.NoError(t, err, "should encode valid YAML")
	actual := lo.Map(decoded.Streams.InputBlacklist, func(rx *regexp2.Regexp, _ int) string {
		return rx.String()
	})
	assert.Exactly(t, patterns, actual, "should keep patterns")
}

func TestInitRegexpRE2Compatible(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	path := filepath.Join(t.TempDir(), "m3u-merge-astra-init-test.yaml")
	cfgBytes := []byte("streams:\n  remove_duplicated_inputs_by_rx_list:\n  - '^https?://(?P<host>[^/]+)'\n" +
		"  input_blacklist:\n  - '^\\d$'\n")
	err := os.WriteFile(path, cfgBytes, 0644)
	assert.NoError(t, err)

	root, _, err := Init(log, path)
	assert.NoError(t, err, "should support named capture groups of regexp package")

	match, err := root.Streams.InputBlacklist[0].MatchString("٣")
	assert.NoError(t, err)
	assert.False(t, match, `\d should match only ASCII digits, as in regexp package`)
}
