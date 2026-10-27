package cli

import (
	"os"
	"testing"

	goFlags "github.com/jessevdk/go-flags"
	pLog "github.com/phuslu/log"
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
	assert.True(t, IsErrOfType(err, goFlags.ErrHelp))

	os.Args = []string{"", "--version"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.True(t, flags.Version)

	os.Args = []string{"", "--logLevel=-1"}
	_, err = Parse()
	assert.Error(t, err, "should return error for negative log level")
	assert.True(t, IsErrOfType(err, goFlags.ErrMarshal))

	os.Args = []string{"", "--logLevel=2"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.Exactly(t, pLog.DebugLevel, flags.LogLevel)

	os.Args = []string{"", "--logLevel=999"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.Exactly(t, pLog.Level(999), flags.LogLevel)

	os.Args = []string{"", "--noninteractive", "--logFile=/log", "--programCfgPath=/cfg/path", "--m3uPath=/m3u/path",
		"--astraAddr=http://127.0.0.1:8005", "--astraUser=admin", "--astraPwd=admin"}
	flags, err = Parse()
	assert.NoError(t, err)
	assert.True(t, flags.Noninteractive)
	assert.Exactly(t, pLog.InfoLevel, flags.LogLevel)
	assert.Exactly(t, "/log", flags.LogFile)
	assert.Exactly(t, "/cfg/path", flags.ProgramCfgPath)
	assert.Exactly(t, "/m3u/path", flags.M3UPath)
	assert.Exactly(t, "http://127.0.0.1:8005", flags.AstraAddr)
	assert.Exactly(t, "admin", flags.AstraUser)
	assert.Exactly(t, "admin", flags.AstraPwd)
}

func TestIsErrOfType(t *testing.T) {
	assert.True(t, IsErrOfType(&goFlags.Error{Type: goFlags.ErrUnknown}, goFlags.ErrUnknown))
	assert.False(t, IsErrOfType(&goFlags.Error{Type: goFlags.ErrUnknown}, goFlags.ErrHelp))
}
