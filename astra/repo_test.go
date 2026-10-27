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

// newDefRepo returns new repository initialized with defaults.
func newDefRepo() repo {
	return NewRepo(logger.New(logger.DebugLevel, os.Stderr), cfg.NewDefCfg())
}

// newDefRepoWithWriter returns new repository initialized with defaults and writing logs to `w`.
func newDefRepoWithWriter(w *syncBuffer) repo {
	return NewRepo(logger.New(logger.DebugLevel, w), cfg.NewDefCfg())
}

func TestNewRepo(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	cfg := cfg.NewDefCfg()

	compareOpt := cmp.FilterPath(func(p cmp.Path) bool {
		return p.Last().String() == ".Formatter"
	}, cmp.Ignore())
	exportedOpt := cmp.Exporter(func(t reflect.Type) bool {
		return true
	})
	reposEqual := cmp.Equal(newDefRepo(), NewRepo(log, cfg), compareOpt, exportedOpt)
	assert.True(t, reposEqual)
}

func TestLog(t *testing.T) {
	compareOpt := cmp.FilterPath(func(p cmp.Path) bool {
		return p.Last().String() == ".Formatter"
	}, cmp.Ignore())
	exportedOpt := cmp.Exporter(func(t reflect.Type) bool {
		return true
	})
	loggersEqual := cmp.Equal(logger.New(logger.DebugLevel, os.Stderr), newDefRepo().Log(), compareOpt, exportedOpt)
	assert.True(t, loggersEqual)
}

func TestCfg(t *testing.T) {
	assert.Exactly(t, cfg.NewDefCfg(), newDefRepo().Cfg())
}
