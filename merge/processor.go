package merge

import (
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
)

// Processor merges M3U channels into astra streams.
//
// Its methods do not modify their arguments.
type Processor struct {
	log *logger.Logger
	cfg cfg.Root
}

// NewProcessor returns new processor logging to `log` and configured with `cfg`.
func NewProcessor(log *logger.Logger, cfg cfg.Root) Processor {
	return Processor{log: log, cfg: cfg}
}
