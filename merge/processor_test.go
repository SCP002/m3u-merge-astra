package merge

import (
	"bytes"
	"os"

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
