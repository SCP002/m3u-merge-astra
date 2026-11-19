package astra

import (
	"time"

	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
)

// Processor processes astra streams and categories.
type Processor struct {
	log              *logger.Logger
	cfg              cfg.Root
	progressInterval time.Duration // How often to log progress of long operations.
}

// NewProcessor returns new processor logging to `log` and configured with `cfg`.
func NewProcessor(log *logger.Logger, cfg cfg.Root) Processor {
	return Processor{log: log, cfg: cfg, progressInterval: time.Second * 30}
}
