package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/example/jev-ai-decision-center/internal/decision"
	"github.com/example/jev-ai-decision-center/internal/domain"
)

type Client struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type apiResponse struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type commonAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Score         *float64           `json:"score"`
	Noul          *float64           `json:"noul"`
	Confidence    *float64           `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

func (c *Client) Decide(ctx context.Context, req decision.Request) (domain.DecisionResult, error) {
	if c.APIKey == "" {
		return domain.DecisionResult{}, errors.New("TYPESAFE_API_KEY is required in live mode")
	}
	start := time.Now()
	body, err := json.Marshal(req)
	if err != nil {
		return domain.DecisionResult{}, err
	}
	url := strings.TrimRight(c.BaseURL, "/") + "/v1/systemone"
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return domain.DecisionResult{}, err
	}
	hreq.Header.Set("Authorization", "Bearer "+c.APIKey)
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(hreq)
	if err != nil {
		return domain.DecisionResult{}, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return domain.DecisionResult{}, fmt.Errorf("jev http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var raw apiResponse
	if err := json.Unmarshal(b, &raw); err != nil {
		return domain.DecisionResult{}, err
	}
	out := domain.DecisionResult{Provider: "jev", Model: raw.Model, Answers: map[string]domain.Answer{}, InputTokens: raw.Usage.InputTokens, OutputTokens: raw.Usage.OutputTokens, LatencyMS: time.Since(start).Milliseconds()}
	for k, v := range raw.Answers {
		var a commonAnswer
		if err := json.Unmarshal(v, &a); err != nil {
			return domain.DecisionResult{}, fmt.Errorf("answer %s: %w", k, err)
		}
		ans := domain.Answer{Type: a.Type, Choice: a.Choice, Score: a.Score, Confidence: a.Confidence, Probabilities: a.Probabilities}
		if a.Noul != nil {
			ans.Probability = a.Noul
		}
		out.Answers[k] = ans
	}
	return out, nil
}

type Mock struct{ Model string }

func (m Mock) Decide(ctx context.Context, req decision.Request) (domain.DecisionResult, error) {
	start := time.Now()
	select {
	case <-ctx.Done():
		return domain.DecisionResult{}, ctx.Err()
	case <-time.After(25 * time.Millisecond):
	}
	b, _ := json.Marshal(req.State)
	s := strings.ToLower(string(b))
	category := "general"
	probs := map[string]float64{"billing": .04, "technical": .07, "infrastructure": .08, "security": .05, "general": .76}
	switch {
	case hasAny(s, "token", "phishing", "suspicious login", "breach", "unauthorized"):
		category = "security"
		probs = map[string]float64{"billing": .01, "technical": .03, "infrastructure": .03, "security": .91, "general": .02}
	case hasAny(s, "database", "outage", "latency", "server", "error rate", "queue", "unavailable"):
		category = "infrastructure"
		probs = map[string]float64{"billing": .01, "technical": .08, "infrastructure": .86, "security": .02, "general": .03}
	case hasAny(s, "bug", "integration", "sdk", "import failed", "api error"):
		category = "technical"
		probs = map[string]float64{"billing": .02, "technical": .89, "infrastructure": .04, "security": .01, "general": .04}
	case hasAny(s, "invoice", "refund", "charge", "subscription", "payment"):
		category = "billing"
		probs = map[string]float64{"billing": .90, "technical": .03, "infrastructure": .02, "security": .01, "general": .04}
	}
	conf := probs[category]
	severity := 1.0
	esc := .18
	action := "route_support"
	if hasAny(s, "critical", "outage", "breach", "many users", "unavailable") {
		severity = 2.8
		esc = .94
	}
	if category == "security" {
		action = "security_review"
	} else if esc >= .9 {
		action = "page_engineer"
	} else if category == "general" {
		action = "human_review"
	}
	return domain.DecisionResult{Provider: "jev-mock", Model: choose(m.Model, "mock-jev"), LatencyMS: time.Since(start).Milliseconds(), Answers: map[string]domain.Answer{
		"category":         {Type: "choice", Choice: category, Confidence: &conf, Probabilities: probs},
		"severity":         {Type: "score", Score: &severity, Confidence: ptr(.88), Probabilities: map[string]float64{"0": .03, "1": .14, "2": .39, "3": .44}},
		"needs_escalation": {Type: "noul", Probability: &esc},
		"next_action":      {Type: "choice", Choice: action, Confidence: ptr(.84), Probabilities: actionProbs(action)},
	}}, nil
}
func hasAny(s string, xs ...string) bool {
	for _, x := range xs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}
func ptr(v float64) *float64 { return &v }
func choose(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func actionProbs(w string) map[string]float64 {
	m := map[string]float64{"observe": .03, "route_support": .06, "page_engineer": .06, "security_review": .04, "human_review": .04}
	m[w] = .81
	return m
}
func ChoiceMargin(a domain.Answer) float64 {
	vals := make([]float64, 0, len(a.Probabilities))
	for _, v := range a.Probabilities {
		vals = append(vals, v)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(vals)))
	if len(vals) < 2 {
		return 0
	}
	return vals[0] - vals[1]
}
