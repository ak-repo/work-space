package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/jev-ai-decision-center/internal/decision"
	"github.com/example/jev-ai-decision-center/internal/domain"
	"github.com/example/jev-ai-decision-center/internal/policy"
	"github.com/example/jev-ai-decision-center/internal/store"
	"github.com/example/jev-ai-decision-center/internal/stream"
)

type Server struct {
	Store    *store.Store
	Provider decision.Provider
	Policy   policy.Engine
	Broker   *stream.Broker
	Model    string
	Timeout  time.Duration
	Web      fs.FS
}

type submitRequest struct {
	Source     string     `json:"source"`
	EventType  string     `json:"event_type"`
	Title      string     `json:"title"`
	State      any        `json:"state"`
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/events", s.submit)
	mux.HandleFunc("GET /api/v1/events", s.recent)
	mux.HandleFunc("GET /api/v1/metrics/summary", s.metrics)
	mux.HandleFunc("GET /api/v1/stream", s.events)
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { jsonOut(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /health/ready", s.ready)
	mux.Handle("/", http.FileServer(http.FS(s.Web)))
	return logging(mux)
}
func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	defer body.Close()
	var in submitRequest
	if err := json.NewDecoder(body).Decode(&in); err != nil {
		jsonOut(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if strings.TrimSpace(in.Title) == "" || in.State == nil {
		jsonOut(w, 422, map[string]string{"error": "title and state are required"})
		return
	}
	if in.Source == "" {
		in.Source = "api"
	}
	if in.EventType == "" {
		in.EventType = "operational_event"
	}
	occurred := time.Now()
	if in.OccurredAt != nil {
		occurred = *in.OccurredAt
	}
	state, _ := json.Marshal(in.State)
	var idem *string
	if v := r.Header.Get("Idempotency-Key"); v != "" {
		idem = &v
	}
	e, err := s.Store.CreateEvent(r.Context(), domain.Event{Source: in.Source, EventType: in.EventType, Title: in.Title, State: state, IdempotencyKey: idem, OccurredAt: occurred})
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": "failed to store event"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()
	start := time.Now()
	d, err := s.Provider.Decide(ctx, decision.Request{State: in.State, Questions: decision.IncidentQuestions(), Model: s.Model})
	if err != nil {
		_ = s.Store.SaveFailedDecision(r.Context(), e.ID, "jev", s.Model, "provider_error", time.Since(start).Milliseconds())
		jsonOut(w, 502, map[string]any{"event_id": e.ID, "error": err.Error()})
		return
	}
	p := s.Policy.Evaluate(d)
	if _, err := s.Store.SaveDecision(r.Context(), e.ID, d, p); err != nil {
		jsonOut(w, 500, map[string]string{"error": "failed to persist decision"})
		return
	}
	out := domain.ProcessResult{Event: e, Decision: d, Policy: p}
	b, _ := json.Marshal(out)
	s.Broker.Publish(b)
	jsonOut(w, 201, out)
}
func (s *Server) recent(w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	x, err := s.Store.Recent(r.Context(), n)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	jsonOut(w, 200, x)
}
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	x, err := s.Store.Metrics(r.Context())
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	jsonOut(w, 200, x)
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Ping(r.Context()); err != nil {
		jsonOut(w, 503, map[string]string{"status": "not-ready"})
		return
	}
	jsonOut(w, 200, map[string]string{"status": "ready"})
}
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	ch := s.Broker.Subscribe()
	defer s.Broker.Unsubscribe(ch)
	fmt.Fprint(w, "event: connected\ndata: {}\n\n")
	fl.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case b := <-ch:
			fmt.Fprintf(w, "event: decision.completed\ndata: %s\n\n", b)
			fl.Flush()
		}
	}
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { start := time.Now(); next.ServeHTTP(w, r); _ = start })
}

var _ = errors.New
var _ = io.EOF
