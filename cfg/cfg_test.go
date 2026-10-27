package cfg

import (
	"m3u-merge-astra/util/copier"
	"m3u-merge-astra/util/file"
	"m3u-merge-astra/util/logger"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
)

func TestSimplifyAliases(t *testing.T) {
	cfg := General{
		NameAliasList: [][]string{
			{"Name 1", "Name 1 var 2", "Name 1 var 3"},
			{"Name_2", "Name_2_var_2"},
		},
	}
	cfgOriginal := copier.TestDeep(t, cfg)

	actual := cfg.SimplifyAliases()

	assert.NotSame(t, &cfg.NameAliasList, &actual, "should return copy of aliases")
	assert.Exactly(t, cfgOriginal, cfg, "should not modify the source config")

	expected := [][]string{
		{"name1", "name1var2", "name1var3"},
		{"name2", "name2var2"},
	}
	assert.Exactly(t, expected, actual, "should simplify aliases")
}

func TestBadRegexpError(t *testing.T) {
	err := error(BadRegexpError{Reason: "Bad", Regexp: *regexp.MustCompile(`.*`)})
	expected := "Bad; Regular expression: .*"
	assert.Exactly(t, expected, err.Error())
}

func TestInitDefault(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	path := filepath.Join(t.TempDir(), "m3u-merge-astra-init-test.yaml")

	// Test creation of the default config.
	actual, isNewCfg, err := Init(log, path)

	assert.Exactly(t, Root{}, actual, "should return empty config")
	assert.True(t, isNewCfg, "should return true")
	assert.NoError(t, err, "should not return error")

	// Test reading of the default config.
	actual, isNewCfg, err = Init(log, path)

	expected := NewDefCfg()

	assert.Exactly(t, expected, actual, "should return default config")
	assert.False(t, isNewCfg, "should return false")
	assert.NoError(t, err, "should not return error")
}

func TestInitValidateCaptureGroups(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	path := filepath.Join(t.TempDir(), "m3u-merge-astra-init-test.yaml")

	// Test reading config with 'remove_duplicated_inputs_by_rx_list'.
	err := file.Copy("init-validate-capture-groups-test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	_, isNewCfg, err := Init(log, path)

	expectedErr := BadRegexpError{
		Regexp: *regexp.MustCompile(`rx_without_capture_group`),
		Reason: "Expecting at least one capture group",
	}

	assert.False(t, isNewCfg, "should return false")
	assert.Exactly(t, expectedErr, errors.UnwrapAll(err), "should return bad regexp error")
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

	assert.Exactly(t, expected, actual.General.SimpleNameAliasList, "actual config should contain these aliases")
	assert.False(t, isNewCfg, "should return false")
	assert.NoError(t, err, "should not return error")
}
