package astra

import (
	"bytes"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
)

// syncBuffer is a concurrency-safe buffer for capturing log output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write implements io.Writer.
func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the buffer contents.
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Reset empties the buffer.
func (b *syncBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

// newDefProcessor returns new processor initialized with defaults.
func newDefProcessor() Processor {
	return NewProcessor(logger.New(logger.DebugLevel, os.Stderr), cfg.NewDefCfg())
}

// newDefProcessorWithWriter returns new processor initialized with defaults and writing logs to `w`.
func newDefProcessorWithWriter(w *syncBuffer) Processor {
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
