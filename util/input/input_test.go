package input

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"m3u-merge-astra/util/logger"

	"github.com/stretchr/testify/assert"
)

// askYesNo runs AskYesNo with input `in`, failing `t` if it does not return in time.
func askYesNo(t *testing.T, log *logger.Logger, in string) bool {
	t.Helper()
	done := make(chan bool, 1)
	go func() { done <- AskYesNo(log, strings.NewReader(in), "prompt\n") }()
	select {
	case answer := <-done:
		return answer
	case <-time.After(time.Second * 5):
		t.Fatalf("AskYesNo did not return for input %q", in)
		return false
	}
}

func TestAskYesNo(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	assert.True(t, askYesNo(t, log, "Y\n"))
	assert.True(t, askYesNo(t, log, "y\n"))
	assert.False(t, askYesNo(t, log, "N\n"))
	assert.False(t, askYesNo(t, log, "n\n"))

	assert.True(t, askYesNo(t, log, "y"), "should accept the last line without newline")
	assert.True(t, askYesNo(t, log, "x\n \nyes\ny\n"), "should ask again, keeping buffered input")
	assert.False(t, askYesNo(t, log, "x\n n \n"), "should ask again, trimming space")

	// Test closed input.
	var out bytes.Buffer
	log = logger.New(logger.DebugLevel, &out)

	assert.False(t, askYesNo(t, log, ""), "should return false on closed input")
	assert.Contains(t, out.String(), "Read from standard input: EOF")

	out.Reset()
	assert.False(t, askYesNo(t, log, "x\nyes"), "should return false on closed input after wrong answers")
	assert.Contains(t, out.String(), "Read from standard input: EOF")
}

func TestAsk(t *testing.T) {
	in := strings.NewReader(" 0 \n")
	answer, err := ask(in, false, "prompt\n", func(s string) bool {
		assert.Exactly(t, " 0 \n", s)
		return false
	})
	assert.NoError(t, err)
	assert.Exactly(t, " 0 \n", answer)

	in.Reset(" 0 \n")
	answer, err = ask(in, true, "prompt\n", func(s string) bool {
		assert.Exactly(t, "0", s)
		return false
	})
	assert.NoError(t, err)
	assert.Exactly(t, "0", answer)

	in.Reset("1\n2\n")
	var inputs []string
	answer, err = ask(in, true, "prompt\n", func(s string) bool {
		inputs = append(inputs, s)
		return true
	})
	assert.ErrorIs(t, err, io.EOF)
	assert.Exactly(t, "", answer)
	assert.Exactly(t, []string{"1", "2", ""}, inputs, "should pass every line, then empty input on EOF")
}
