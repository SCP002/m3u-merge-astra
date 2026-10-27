package input

import (
	"os"
	"strings"
	"testing"

	"m3u-merge-astra/util/logger"

	"github.com/stretchr/testify/assert"
)

func TestAskYesNo(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	in := strings.NewReader("Y\n")
	answer := AskYesNo(log, in, "prompt\n")
	assert.True(t, answer)

	in.Reset("y\n")
	answer = AskYesNo(log, in, "prompt\n")
	assert.True(t, answer)

	in.Reset("N\n")
	answer = AskYesNo(log, in, "prompt\n")
	assert.False(t, answer)

	in.Reset("n\n")
	answer = AskYesNo(log, in, "prompt\n")
	assert.False(t, answer)
}

func TestAsk(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	in := strings.NewReader(" 0 \n")
	answer := ask(log, in, false, "prompt\n", func(s string) bool {
		assert.Exactly(t, " 0 \n", s)
		return false
	})
	assert.Exactly(t, " 0 \n", answer)

	in.Reset(" 0 \n")
	answer = ask(log, in, true, "prompt\n", func(s string) bool {
		assert.Exactly(t, "0", s)
		return false
	})
	assert.Exactly(t, "0", answer)
}
