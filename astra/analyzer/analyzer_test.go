package analyzer

import (
	"encoding/json/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"m3u-merge-astra/util/logger"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestNew(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	analyzer := New(log, "127.0.0.1", time.Second)
	assert.Exactly(t, log, analyzer.log)
	assert.Exactly(t, "ws://127.0.0.1/api/", analyzer.url)
	assert.Exactly(t, time.Second, analyzer.dialer.HandshakeTimeout)
}

// newHandlerServer starts analyzer server closed on `t` cleanup, which runs `handle` for every connection and closes
// the connection afterwards.
func newHandlerServer(t *testing.T, handle func(conn *websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		handle(conn)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// fakeServer emulates astra analyzer websocket API (astra --analyze).
//
// After start request it periodically sends a message produced by `respond` for requested address.
// Like real analyzer, it keeps the connection open until the client closes it.
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
	s.Server = newHandlerServer(t, func(conn *websocket.Conn) {
		var req startReq
		_, reqBytes, err := conn.ReadMessage()
		if err != nil || json.Unmarshal(reqBytes, &req) != nil || req.Cmd != "start" {
			return
		}
		s.mu.Lock()
		s.starts[req.Address]++
		s.mu.Unlock()

		// Read stop request until the client closes the connection.
		closed := make(chan struct{})
		go func() {
			defer close(closed)
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if strings.Contains(string(msg), `"stop"`) {
					s.mu.Lock()
					s.stops[req.Address]++
					s.mu.Unlock()
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
			case <-closed:
				return
			}
		}
	})
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

// assertStops asserts that the server soon gets `expected` amount of stop requests for `address`, as client does not
// wait for the server to handle it.
func (s *fakeServer) assertStops(t *testing.T, address string, expected int, msgAndArgs ...any) {
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		_, stops := s.counts(address)
		assert.Exactly(c, expected, stops)
	}, time.Second, time.Millisecond*10, msgAndArgs...)
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
	t.Run("checks", func(t *testing.T) {
		t.Run("alive", func(t *testing.T) {
			t.Parallel()
			result, err := analyzer.Check(watchTime, maxAttempts, aliveURL)
			require.NoError(t, err)
			expected := Result{Bitrate: 1500, CCErrors: 3, PCRErrors: 2, PESErrors: 1, Scrambled: true,
				HasAudio: true, HasVideo: true}
			assert.Exactly(t, expected, result)
			starts, _ := server.counts(aliveURL)
			assert.Exactly(t, 1, starts, "should stop trying after getting bitrate")
			server.assertStops(t, aliveURL, 1, "should send stop request")
		})

		t.Run("dead", func(t *testing.T) {
			t.Parallel()
			result, err := analyzer.Check(watchTime, maxAttempts, deadURL)
			require.NoError(t, err, "dead input is not an error, bitrate should be checked instead")
			assert.Exactly(t, Result{}, result)
			starts, _ := server.counts(deadURL)
			assert.Exactly(t, maxAttempts, starts, "should retry while bitrate is 0")
			server.assertStops(t, deadURL, maxAttempts)
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
}

func TestCheckInFlightResponses(t *testing.T) {
	// Responses are sent non-stop, so reader always has the next one when the client stops collecting them.
	server := newHandlerServer(t, func(conn *websocket.Conn) {
		for conn.WriteMessage(websocket.TextMessage, []byte(`{"total": {"bitrate": 1}}`)) == nil {
		}
	})

	analyzer := New(logger.New(logger.DebugLevel, os.Stderr), server.Listener.Addr().String(), time.Second)
	// Blocked reader is reported by goleak in TestMain.
	_, err := analyzer.Check(time.Millisecond, 1, "http://url")
	assert.NoError(t, err)
}
