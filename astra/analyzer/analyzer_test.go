package analyzer

import (
	"bytes"
	"encoding/json/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	"m3u-merge-astra/util/logger"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	analyzer := New(log, "127.0.0.1", time.Second)
	assert.Exactly(t, log, analyzer.log)
	assert.Exactly(t, "ws://127.0.0.1/api/", analyzer.url)
	assert.Exactly(t, time.Second, analyzer.dialer.HandshakeTimeout)
	assert.Exactly(t, time.Second, analyzer.closeTimeout)
}

// fakeServer emulates astra analyzer websocket API (astra --analyze).
//
// After start request it periodically sends a message produced by `respond` for requested address.
// Like real analyzer, it ignores close message and keeps the connection open until the client closes it.
type fakeServer struct {
	*httptest.Server
	respond func(address string) []byte
	mu      sync.Mutex
	starts  map[string]int // Amount of start requests per address.
	stops   map[string]int // Amount of stop requests per address.
}

// newFakeServer starts fake analyzer server closed on `t` cleanup.
func newFakeServer(t *testing.T, respond func(address string) []byte) *fakeServer {
	s := &fakeServer{respond: respond, starts: map[string]int{}, stops: map[string]int{}}
	upgrader := websocket.Upgrader{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		conn.SetCloseHandler(func(int, string) error { return nil }) // Do not reply to close message.

		var req startReq
		_, reqBytes, err := conn.ReadMessage()
		if err != nil || json.Unmarshal(reqBytes, &req) != nil || req.Cmd != "start" {
			return
		}
		s.mu.Lock()
		s.starts[req.Address]++
		s.mu.Unlock()

		// Read stop request and close message.
		stopped := make(chan struct{})
		closed := make(chan struct{})
		go func() {
			defer close(closed)
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return // Close message or broken connection.
				}
				if strings.Contains(string(msg), `"stop"`) {
					s.mu.Lock()
					s.stops[req.Address]++
					s.mu.Unlock()
					close(stopped)
				}
			}
		}()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"cmd": "start"}`))
		ticker := time.NewTicker(time.Millisecond * 10)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if conn.WriteMessage(websocket.TextMessage, s.respond(req.Address)) != nil {
					return
				}
			case <-stopped:
				<-closed
				// Wait for the client to close the connection.
				_, _ = conn.NetConn().Read(make([]byte, 1))
				return
			case <-closed:
				return
			}
		}
	})

	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

// addr returns server address in format of 'host:port'.
func (s *fakeServer) addr() string {
	return s.Listener.Addr().String()
}

// counts returns amount of start and stop requests for `address`.
func (s *fakeServer) counts(address string) (starts, stops int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starts[address], s.stops[address]
}

func TestCheck(t *testing.T) {
	const (
		aliveURL   = "http://alive"
		deadURL    = "http://dead"
		invalidURL = "http://invalid"
	)
	server := newFakeServer(t, func(address string) []byte {
		switch address {
		case aliveURL:
			return []byte(`{"on_air": true, "total": {"bitrate": 1500, "cc_errors": 3, "pcr_errors": 2, ` +
				`"pes_errors": 1, "scrambled": true}, "streams": [{"type_name": "VIDEO"}, {"type_name": "AUDIO"}]}`)
		case deadURL:
			return []byte(`{"on_air": false, "total": {"bitrate": 0}}`)
		default:
			return []byte(`not json`)
		}
	})

	log := logger.New(logger.DebugLevel, os.Stderr)
	watchTime := time.Millisecond * 100
	maxAttempts := 3

	// Checks run in parallel against the same analyzer.
	analyzer := New(log, server.addr(), time.Second)
	analyzer.closeTimeout = time.Millisecond * 50
	t.Run("checks", func(t *testing.T) {
		t.Run("alive", func(t *testing.T) {
			t.Parallel()
			result, err := analyzer.Check(watchTime, maxAttempts, aliveURL)
			require.NoError(t, err)
			expected := Result{Bitrate: 1500, CCErrors: 3, PCRErrors: 2, PESErrors: 1, Scrambled: true,
				HasAudio: true, HasVideo: true}
			assert.Exactly(t, expected, result)
			starts, stops := server.counts(aliveURL)
			assert.Exactly(t, 1, starts, "should stop trying after getting bitrate")
			assert.Exactly(t, 1, stops, "should send stop request")
		})

		t.Run("dead", func(t *testing.T) {
			t.Parallel()
			result, err := analyzer.Check(watchTime, maxAttempts, deadURL)
			require.NoError(t, err, "dead input is not an error, bitrate should be checked instead")
			assert.Exactly(t, Result{}, result)
			starts, stops := server.counts(deadURL)
			assert.Exactly(t, maxAttempts, starts, "should retry while bitrate is 0")
			assert.Exactly(t, maxAttempts, stops)
		})

		t.Run("invalid response", func(t *testing.T) {
			t.Parallel()
			result, err := analyzer.Check(watchTime, maxAttempts, invalidURL)
			assert.ErrorContains(t, err, "Decode response")
			assert.Exactly(t, Result{}, result)
			starts, _ := server.counts(invalidURL)
			assert.Exactly(t, 1, starts, "should not retry on error")
		})

		t.Run("analyzer is unreachable", func(t *testing.T) {
			t.Parallel()
			// Reserve a free port and release it, so connections to it get refused.
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			addr := listener.Addr().String()
			_ = listener.Close()

			result, err := New(log, addr, time.Second).Check(watchTime, maxAttempts, aliveURL)
			assert.ErrorContains(t, err, "Dial")
			assert.Exactly(t, Result{}, result)
		})
	})

	// Every check should clean up its goroutines.
	var leaks bytes.Buffer
	leakProfile := pprof.Lookup("goroutineleak")
	require.NoError(t, leakProfile.WriteTo(&leaks, 1))
	assert.Zero(t, leakProfile.Count(), "goroutines leaked:\n%s", leaks.String())
}

func TestNewFake(t *testing.T) {
	assert.Exactly(t, &fakeAnalyzer{urlResultMap: map[string]Result{}}, NewFake())
}

func TestAddResult(t *testing.T) {
	analyzer := NewFake()
	analyzer.AddResult("url1", Result{Bitrate: 1})
	analyzer.AddResult("url2", Result{Bitrate: 2})

	expected := map[string]Result{"url1": {Bitrate: 1}, "url2": {Bitrate: 2}}
	assert.Exactly(t, expected, analyzer.urlResultMap)
}

func TestFakeCheck(t *testing.T) {
	analyzer := NewFake()
	analyzer.AddResult("url1", Result{Bitrate: 1})

	result, err := analyzer.Check(time.Second, 1, "url1")
	assert.Exactly(t, Result{Bitrate: 1}, result)
	assert.NoError(t, err)
}
