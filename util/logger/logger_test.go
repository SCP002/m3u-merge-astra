package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var timeRx = `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Trace("message")
	log.Debug("message")
	out := buf.String()
	assert.NotRegexp(t, regexp.MustCompile(`TRACE`), out, "should not print trace messages with debug level logger")
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG message\n`), out)
}

func TestLevels(t *testing.T) {
	levels := []struct {
		name   string
		plain  func(Logger, any)
		format func(Logger, string, ...any)
		withFi func(Logger, string, ...any)
		panics bool
		exits  bool
	}{
		{name: "TRACE", plain: Logger.Trace, format: Logger.Tracef, withFi: Logger.TraceFi},
		{name: "DEBUG", plain: Logger.Debug, format: Logger.Debugf, withFi: Logger.DebugFi},
		{name: "INFO", plain: Logger.Info, format: Logger.Infof, withFi: Logger.InfoFi},
		{name: "WARN", plain: Logger.Warn, format: Logger.Warnf, withFi: Logger.WarnFi},
		{name: "ERROR", plain: Logger.Error, format: Logger.Errorf, withFi: Logger.ErrorFi},
		{name: "PANIC", plain: Logger.Panic, format: Logger.Panicf, withFi: Logger.PanicFi, panics: true},
		{name: "FATAL", plain: Logger.Fatal, format: Logger.Fatalf, withFi: Logger.FatalFi, exits: true},
	}
	for _, lvl := range levels {
		calls := []struct {
			name string
			call func(Logger)
			want string
		}{
			{name: "plain", call: func(l Logger) { lvl.plain(l, "message") }, want: `message`},
			{name: "format", call: func(l Logger) { lvl.format(l, "%v + %v", "message 1", "message 2") },
				want: `message 1 \+ message 2`},
			{name: "fields", call: func(l Logger) { lvl.withFi(l, "message", "a", "b", 1, 2) },
				want: `message: a "b", 1 "2"`},
			{name: "no fields", call: func(l Logger) { lvl.withFi(l, "message") }, want: `message`},
		}
		for _, c := range calls {
			t.Run(lvl.name+"/"+c.name, func(t *testing.T) {
				wantRx := regexp.MustCompile(timeRx + ` ` + lvl.name + ` ` + c.want + `\n`)
				if lvl.exits {
					assertExits(t, func() { c.call(*New(TraceLevel, os.Stderr)) }, wantRx)
					return
				}
				var buf bytes.Buffer
				log := New(TraceLevel, &buf)
				if lvl.panics {
					assert.Panics(t, func() { c.call(*log) })
				} else {
					c.call(*log)
				}
				assert.Regexp(t, wantRx, buf.String())
			})
		}
	}
}

// assertExits runs `fn` in a new process of the current test and asserts it exits with code 255 and output matching
// `wantRx`.
func assertExits(t *testing.T, fn func(), wantRx *regexp.Regexp) {
	t.Helper()
	const env = "LOGGER_TEST_EXIT"
	if os.Getenv(env) == "1" {
		fn()
		return
	}
	// Anchor every part of the name, as -test.run matches every part of the subtest name separately.
	parts := strings.Split(t.Name(), "/")
	for i, part := range parts {
		parts[i] = "^" + regexp.QuoteMeta(part) + "$"
	}
	cmd := exec.Command(os.Args[0], "-test.run="+strings.Join(parts, "/"))
	cmd.Env = append(os.Environ(), env+"=1")
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if assert.ErrorAs(t, err, &exitErr, "should exit with error") {
		assert.Exactly(t, 255, exitErr.ExitCode())
	}
	assert.Regexp(t, wantRx, string(out))
}

func TestAddFileWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	log := New(DebugLevel, os.Stderr)
	file, err := log.AddFileWriter("")
	assert.NoError(t, err, "should not return error for empty file path")
	assert.Nil(t, file, "should return nil file for empty file path")

	file, err = log.AddFileWriter(path)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.FileExists(t, path)

	log.InfoFi("message 1", "field1", "value 1", "field2", 2)
	log.WarnFi("message 2", "field1", "value 3", "field2", 4)

	// file.Sync() does not help, content is empty, closing and opening the file again.
	_ = file.Close()
	file, err = os.Open(path)
	assert.NoError(t, err)
	defer func() { _ = file.Close() }()
	reader := bufio.NewReader(file)

	type logEntry struct {
		Time    string `json:"time"`
		Level   string `json:"level"`
		Field1  string `json:"field1"`
		Field2  int    `json:"field2"`
		Message string `json:"message"`
	}

	var entry logEntry

	line, err := reader.ReadBytes('\n')
	assert.NoError(t, err)
	err = json.Unmarshal(line, &entry)
	assert.NoError(t, err)
	_, err = time.Parse(time.RFC3339Nano, entry.Time)
	assert.NoError(t, err, "time in entry must be in RFC3339 format with millis")
	assert.Exactly(t, "info", entry.Level, "log severity must be that level")
	assert.Exactly(t, "value 1", entry.Field1)
	assert.Exactly(t, 2, entry.Field2)
	assert.Exactly(t, "message 1", entry.Message)

	line, err = reader.ReadBytes('\n')
	assert.NoError(t, err)
	err = json.Unmarshal(line, &entry)
	assert.NoError(t, err)
	_, err = time.Parse(time.RFC3339Nano, entry.Time)
	assert.NoError(t, err, "time in entry must be in RFC3339 format with millis")
	assert.Exactly(t, "warn", entry.Level, "log severity must be that level")
	assert.Exactly(t, "value 3", entry.Field1)
	assert.Exactly(t, 4, entry.Field2)
	assert.Exactly(t, "message 2", entry.Message)
}
