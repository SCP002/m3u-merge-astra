package m3u

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
)

// newDefProcessor returns new processor initialized with defaults.
func newDefProcessor() Processor {
	return NewProcessor(logger.New(logger.DebugLevel, os.Stderr), cfg.NewDefCfg())
}

// newDefProcessorWithWriter returns new processor initialized with defaults and writing logs to `w`.
func newDefProcessorWithWriter(w *bytes.Buffer) Processor {
	return NewProcessor(logger.New(logger.DebugLevel, w), cfg.NewDefCfg())
}

func TestNewProcessor(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	cfg := cfg.NewDefCfg()

	compareOpt := cmp.FilterPath(func(p cmp.Path) bool {
		return p.Last().String() == ".Formatter"
	}, cmp.Ignore())
	exportedOpt := cmp.Exporter(func(t reflect.Type) bool {
		return true
	})
	processorsEqual := cmp.Equal(newDefProcessor(), NewProcessor(log, cfg), compareOpt, exportedOpt)
	assert.True(t, processorsEqual)
}
