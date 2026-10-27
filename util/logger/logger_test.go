package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	pLog "github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
)

var timeRx = `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Trace("message")
	log.Debug("message")
	log.DebugFi("message", "k", "v")
	log.InfoFi("message", "1", 2)
	log.Warn("message")
	log.Error("message")
	assert.Panics(t, func() { log.Panic("message") })
	out := buf.String()
	assert.NotRegexp(t, regexp.MustCompile(`TRACE`), out, "should not print trace messages with debug level logger")
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG message: k "v"`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message: 1 "2"`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message`), out)
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message`), out)
}

func TestTrace(t *testing.T) {
	var buf bytes.Buffer
	log := New(TraceLevel, &buf)
	log.Trace("message")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` TRACE message`), out)
}

func TestTracef(t *testing.T) {
	var buf bytes.Buffer
	log := New(TraceLevel, &buf)
	log.Tracef("%v + %v", "message 1", "message 2")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` TRACE message 1 \+ message 2`), out)
}

func TestTraceFi(t *testing.T) {
	var buf bytes.Buffer
	log := New(TraceLevel, &buf)
	log.TraceFi("message", "a", "b", 1, 2)
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` TRACE message: a "b", 1 "2"`), out)
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Debug("message")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG message`), out)
}

func TestDebugf(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Debugf("%v + %v", "message 1", "message 2")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG message 1 \+ message 2`), out)
}

func TestDebugFi(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.DebugFi("message", "a", "b", 1, 2)
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` DEBUG message: a "b", 1 "2"`), out)
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Info("message")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message`), out)
}

func TestInfof(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Infof("%v + %v", "message 1", "message 2")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message 1 \+ message 2`), out)
}

func TestInfoFi(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.InfoFi("message", "a", "b", 1, 2)
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message: a "b", 1 "2"`), out)
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Warn("message")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message`), out)
}

func TestWarnf(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Warnf("%v + %v", "message 1", "message 2")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message 1 \+ message 2`), out)
}

func TestWarnFi(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.WarnFi("message", "a", "b", 1, 2)
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` WARN message: a "b", 1 "2"`), out)
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Error("message")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message`), out)
}

func TestErrorf(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.Errorf("%v + %v", "message 1", "message 2")
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message 1 \+ message 2`), out)
}

func TestErrorFi(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	log.ErrorFi("message", "a", "b", 1, 2)
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` ERROR message: a "b", 1 "2"`), out)
}

func TestFatal(t *testing.T) {
	if os.Getenv("TestFatal") == "1" {
		log := New(DebugLevel, os.Stderr)
		log.Fatal("message")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatal$")
	cmd.Env = append(os.Environ(), "TestFatal=1")
	out, err := cmd.CombinedOutput()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		assert.Exactly(t, 255, e.ExitCode())
		assert.Regexp(t, regexp.MustCompile(timeRx+` FATAL message`), string(out))
		return
	}
	assert.Failf(t, "", "expected exec.ExitError, got '%v'", err)
}

func TestFatalf(t *testing.T) {
	if os.Getenv("TestFatalf") == "1" {
		log := New(DebugLevel, os.Stderr)
		log.Fatalf("%v + %v", "message 1", "message 2")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatalf$")
	cmd.Env = append(os.Environ(), "TestFatalf=1")
	out, err := cmd.CombinedOutput()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		assert.Exactly(t, 255, e.ExitCode())
		assert.Regexp(t, regexp.MustCompile(timeRx+` FATAL message 1 \+ message 2`), string(out))
		return
	}
	assert.Failf(t, "", "expected exec.ExitError, got '%v'", err)
}

func TestFatalFi(t *testing.T) {
	if os.Getenv("TestFatalFi") == "1" {
		log := New(DebugLevel, os.Stderr)
		log.FatalFi("message", "a", "b", 1, 2)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatalFi$")
	cmd.Env = append(os.Environ(), "TestFatalFi=1")
	out, err := cmd.CombinedOutput()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		assert.Exactly(t, 255, e.ExitCode())
		assert.Regexp(t, regexp.MustCompile(timeRx+` FATAL message: a "b", 1 "2"`), string(out))
		return
	}
	assert.Failf(t, "", "expected exec.ExitError, got '%v'", err)
}

func TestPanic(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	assert.Panics(t, func() { log.Panic("message") })
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message`), out)
}

func TestPanicf(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	assert.Panics(t, func() { log.Panicf("%v + %v", "message 1", "message 2") })
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message 1 \+ message 2`), out)
}

func TestPanicFi(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	assert.Panics(t, func() { log.PanicFi("message", "a", "b", 1, 2) })
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` PANIC message: a "b", 1 "2"`), out)
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
	file.Close()
	file, err = os.Open(path)
	assert.NoError(t, err)
	defer file.Close()
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

func TestPrint(t *testing.T) {
	var buf bytes.Buffer
	log := New(DebugLevel, &buf)
	print(log.Logger.Info(), "message", nil)
	out := buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message`), out)

	buf.Reset()
	log = New(DebugLevel, &buf)
	print(log.Logger.Info(), "message", []any{"a", "b", 1, 2})
	out = buf.String()
	assert.Regexp(t, regexp.MustCompile(timeRx+` INFO message: a "b", 1 "2"`), out)
}

func TestNewConsoleFormatter(t *testing.T) {
	// Tested in TestNew.
	var formatter func(io.Writer, *pLog.FormatterArgs) (int, error)
	assert.IsType(t, formatter, newConsoleFormatter(false, ""))
}
