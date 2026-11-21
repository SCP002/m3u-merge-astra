package cli

import (
	"os"
	"testing"

	"m3u-merge-astra/util/logger"

	goflags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	os.Args = []string{""}
	flags, err := Parse()
	assert.NoError(t, err)
	assert.False(t, flags.Noninteractive, "noninteracive flag should be false if not specified")
	assert.Empty(t, flags.LogFile, "logFile flag should be empty if not specified")

	os.Args = []string{"", "--help"}
	_, err = Parse()
	assert.True(t, IsErrOfType(err, goflags.ErrHelp))

	os.Args = []string{"", "--version"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.True(t, flags.Version)

	os.Args = []string{"", "--update"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.True(t, flags.Update)

	os.Args = []string{"", "--logLevel=-1"}
	_, err = Parse()
	assert.Error(t, err, "should return error for negative log level")
	assert.True(t, IsErrOfType(err, goflags.ErrMarshal))

	os.Args = []string{"", "--logLevel=2"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.Exactly(t, logger.DebugLevel, flags.LogLevel)

	os.Args = []string{"", "--logLevel=1"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.Exactly(t, logger.TraceLevel, flags.LogLevel)

	os.Args = []string{"", "--logLevel=7"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.Exactly(t, logger.PanicLevel, flags.LogLevel)

	for _, lvl := range []string{"0", "8", "999"} {
		os.Args = []string{"", "--logLevel=" + lvl}
		_, err = Parse()
		assert.ErrorContains(t, err, "invalid log level "+lvl+", must be from 1 to 7")
		assert.True(t, IsErrOfType(err, goflags.ErrMarshal))
	}

	os.Args = []string{"", "--noninteractive", "--logFile=/log", "--programCfgPath=/cfg/path", "--m3uPath=/m3u/path",
		"--astraAddr=http://127.0.0.1:8005", "--astraUser=admin", "--astraPwd=admin"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.True(t, flags.Noninteractive)
	assert.Exactly(t, logger.InfoLevel, flags.LogLevel)
	assert.Exactly(t, "/log", flags.LogFile)
	assert.Exactly(t, "/cfg/path", flags.ProgramCfgPath)
	assert.Exactly(t, "/m3u/path", flags.M3UPath)
	assert.Exactly(t, "http://127.0.0.1:8005", flags.AstraAddr)
	assert.Exactly(t, "admin", flags.AstraUser)
	assert.Exactly(t, "admin", flags.AstraPwd)
}

func TestIsErrOfType(t *testing.T) {
	assert.True(t, IsErrOfType(&goflags.Error{Type: goflags.ErrUnknown}, goflags.ErrUnknown))
	assert.False(t, IsErrOfType(&goflags.Error{Type: goflags.ErrUnknown}, goflags.ErrHelp))
}
