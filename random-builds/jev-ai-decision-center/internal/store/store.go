package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/example/jev-ai-decision-center/internal/domain"
)

type persistedDecision struct {
	Event     domain.Event           `json:"event"`
	Decision  *domain.DecisionResult `json:"decision,omitempty"`
	Policy    *domain.PolicyOutcome  `json:"policy,omitempty"`
	Status    string                 `json:"status"`
	ErrorCode string                 `json:"error_code,omitempty"`
}

type snapshot struct {
	Items []persistedDecision `json:"items"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	byID  map[string]*persistedDecision
	order []string
	idem  map[string]string
}

func Open(_ context.Context, path string) (*Store, error) {
	s := &Store{path: path, byID: map[string]*persistedDecision{}, idem: map[string]string{}}
	if path == "" {
		path = "data/state.json"
		s.path = path
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store) Close()                     {}
func (s *Store) Ping(context.Context) error { return nil }

func (s *Store) CreateEvent(_ context.Context, e domain.Event) (domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.IdempotencyKey != nil {
		if id, ok := s.idem[*e.IdempotencyKey]; ok {
			return s.byID[id].Event, nil
		}
	}
	e.ID = newID()
	e.CreatedAt = time.Now().UTC()
	if e.OccurredAt.IsZero() {
		e.OccurredAt = e.CreatedAt
	}
	p := &persistedDecision{Event: e, Status: "pending"}
	s.byID[e.ID] = p
	s.order = append(s.order, e.ID)
	if e.IdempotencyKey != nil {
		s.idem[*e.IdempotencyKey] = e.ID
	}
	return e, s.persistLocked()
}
func (s *Store) SaveDecision(_ context.Context, eventID string, d domain.DecisionResult, p domain.PolicyOutcome) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.byID[eventID]
	if !ok {
		return "", errors.New("event not found")
	}
	it.Decision = &d
	it.Policy = &p
	it.Status = "ok"
	it.ErrorCode = ""
	return newID(), s.persistLocked()
}
func (s *Store) SaveFailedDecision(_ context.Context, eventID, provider, model, code string, latency int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.byID[eventID]
	if !ok {
		return errors.New("event not found")
	}
	it.Decision = &domain.DecisionResult{Provider: provider, Model: model, LatencyMS: latency}
	it.Status = "failed"
	it.ErrorCode = code
	return s.persistLocked()
}
func (s *Store) Recent(_ context.Context, limit int) ([]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 || limit > 200 {
		limit = 50
	}
	out := []map[string]any{}
	for i := len(s.order) - 1; i >= 0 && len(out) < limit; i-- {
		it := s.byID[s.order[i]]
		row := map[string]any{"id": it.Event.ID, "title": it.Event.Title, "event_type": it.Event.EventType, "created_at": it.Event.CreatedAt}
		if it.Decision != nil {
			row["provider"] = it.Decision.Provider
			row["model"] = it.Decision.Model
			row["latency_ms"] = it.Decision.LatencyMS
			if a, ok := it.Decision.Answers["category"]; ok {
				row["category"] = a.Choice
				if a.Confidence != nil {
					row["confidence"] = *a.Confidence
				}
			}
		}
		if it.Policy != nil {
			row["action"] = it.Policy.Action
			row["reason_code"] = it.Policy.ReasonCode
		}
		out = append(out, row)
	}
	return out, nil
}
func (s *Store) Metrics(_ context.Context) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	actions := map[string]int{}
	var total int
	var sum int64
	for _, id := range s.order {
		it := s.byID[id]
		if it.Status == "ok" && it.Decision != nil {
			total++
			sum += it.Decision.LatencyMS
			if it.Policy != nil {
				actions[it.Policy.Action]++
			}
		}
	}
	var avg any = nil
	if total > 0 {
		avg = float64(sum) / float64(total)
	}
	return map[string]any{"decisions": total, "avg_latency_ms": avg, "actions": actions}, nil
}
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var snap snapshot
	if len(b) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	sort.SliceStable(snap.Items, func(i, j int) bool { return snap.Items[i].Event.CreatedAt.Before(snap.Items[j].Event.CreatedAt) })
	for i := range snap.Items {
		it := snap.Items[i]
		cp := it
		s.byID[it.Event.ID] = &cp
		s.order = append(s.order, it.Event.ID)
		if it.Event.IdempotencyKey != nil {
			s.idem[*it.Event.IdempotencyKey] = it.Event.ID
		}
	}
	return nil
}
func (s *Store) persistLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	snap := snapshot{Items: make([]persistedDecision, 0, len(s.order))}
	for _, id := range s.order {
		snap.Items = append(snap.Items, *s.byID[id])
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
func newID() string { var b [16]byte; _, _ = rand.Read(b[:]); return hex.EncodeToString(b[:]) }
