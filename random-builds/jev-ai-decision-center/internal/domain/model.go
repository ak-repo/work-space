package domain

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID             string          `json:"id"`
	Source         string          `json:"source"`
	EventType      string          `json:"event_type"`
	Title          string          `json:"title"`
	State          json.RawMessage `json:"state"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	OccurredAt     time.Time       `json:"occurred_at"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Probability   *float64           `json:"probability,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
}

type DecisionResult struct {
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	Answers      map[string]Answer `json:"answers"`
	InputTokens  int               `json:"input_tokens,omitempty"`
	OutputTokens int               `json:"output_tokens,omitempty"`
	LatencyMS    int64             `json:"latency_ms"`
}

type PolicyOutcome struct {
	Action     string         `json:"action"`
	ReasonCode string         `json:"reason_code"`
	Details    map[string]any `json:"details,omitempty"`
}

type ProcessResult struct {
	Event    Event          `json:"event"`
	Decision DecisionResult `json:"decision"`
	Policy   PolicyOutcome  `json:"policy"`
}
