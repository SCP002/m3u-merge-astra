// Package apitest provides fake astra control API for tests.
package apitest

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

// User and Password are credentials accepted by the fake.
const (
	User     = "admin"
	Password = "admin"
)

// Server is a fake astra control API, mimicing:
//   - set-stream removes stream with requested ID if exists, then appends the passed stream object as is, unless it
//     has "remove" field set to true. Object is stored verbatim, so its "id" field may differ from the requested one;
//   - set-category replaces category at requested index with passed object as is, appends it if index is not set or
//     removes category at index if object has "remove" field set to true;
//   - empty lists are omitted from "load" response.
type Server struct {
	*httptest.Server
	t          testing.TB
	mu         sync.Mutex
	streams    []jsontext.Value
	categories []jsontext.Value
}

// NewServer starts new fake astra control API with empty config, closed on `t` cleanup.
func NewServer(t testing.TB) *Server {
	s := &Server{t: t}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /control/", s.handleControl)
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

// request represents any request to control API.
type request struct {
	Cmd      string         `json:"cmd"`
	ID       jsontext.Value `json:"id"`
	Stream   jsontext.Value `json:"stream"`
	Category jsontext.Value `json:"category"`
}

// removable represents fields of stream or category object used by the fake.
type removable struct {
	ID     *string `json:"id"`
	Remove bool    `json:"remove"`
}

// handleControl handles requests to /control/ endpoint.
func (s *Server) handleControl(w http.ResponseWriter, r *http.Request) {
	if user, password, ok := r.BasicAuth(); !ok || user != User || password != Password {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusNotFound) // Astra responds with 404 to malformed requests.
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch req.Cmd {
	case "load":
		// Astra omits empty lists.
		cfg := map[string]any{"settings": map[string]any{}}
		if len(s.streams) > 0 {
			cfg["make_stream"] = s.streams
		}
		if len(s.categories) > 0 {
			cfg["categories"] = s.categories
		}
		s.respond(w, cfg)
	case "sessions":
		s.respond(w, map[string]any{"sessions": []any{}})
	case "set-stream":
		s.setStream(w, req)
	case "set-category":
		s.setCategory(w, req, body)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// setStream handles set-stream command.
func (s *Server) setStream(w http.ResponseWriter, req request) {
	var id string
	var stream removable
	if json.Unmarshal(req.ID, &id) != nil || json.Unmarshal(req.Stream, &stream) != nil {
		s.respond(w, map[string]string{"set-stream": "er"})
		return
	}

	s.streams = slices.DeleteFunc(s.streams, func(v jsontext.Value) bool {
		var stored removable
		_ = json.Unmarshal(v, &stored)
		return stored.ID != nil && *stored.ID == id
	})
	if !stream.Remove {
		s.streams = append(s.streams, req.Stream)
	}
	s.respond(w, map[string]string{"set-stream": "ok"})
}

// setCategory handles set-category command.
func (s *Server) setCategory(w http.ResponseWriter, req request, body []byte) {
	var category removable
	if json.Unmarshal(req.Category, &category) != nil {
		s.crash(body)
		return
	}

	if len(req.ID) == 0 || req.ID.Kind() == 'n' {
		if category.Remove {
			s.crash(body)
			return
		}
		s.categories = append(s.categories, req.Category)
		s.respond(w, map[string]string{"set-category": "ok"})
		return
	}

	var idx int
	if json.Unmarshal(req.ID, &idx) != nil || idx < 0 || idx >= len(s.categories) {
		s.crash(body)
		return
	}
	if category.Remove {
		s.categories = slices.Delete(s.categories, idx, idx+1)
	} else {
		s.categories[idx] = req.Category
	}
	s.respond(w, map[string]string{"set-category": "ok"})
}

// crash fails the test as request with `body` would crash real astra and closes the connection without response.
func (s *Server) crash(body []byte) {
	s.t.Errorf("Request would crash real astra: %s", body)
	panic(http.ErrAbortHandler)
}

// respond writes `v` as JSON response.
func (s *Server) respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, v); err != nil {
		s.t.Errorf("Encode fake astra response: %v", err)
	}
}
