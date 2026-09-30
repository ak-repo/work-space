package policy

import (
	"github.com/example/jev-ai-decision-center/internal/config"
	"github.com/example/jev-ai-decision-center/internal/domain"
	"testing"
)

func TestSecurityAlwaysRoutesToSecurityReviewAfterConfidenceGates(t *testing.T) {
	c := .92
	d := domain.DecisionResult{Answers: map[string]domain.Answer{"category": {Type: "choice", Choice: "security", Confidence: &c, Probabilities: map[string]float64{"security": .92, "general": .03}}}}
	got := (Engine{Cfg: config.PolicyConfig{MinCategoryConfidence: .75, MinChoiceMargin: .15, EscalationProbability: .9}}).Evaluate(d)
	if got.Action != "security_review" {
		t.Fatalf("got %s", got.Action)
	}
}
func TestLowConfidenceNeedsHuman(t *testing.T) {
	c := .60
	d := domain.DecisionResult{Answers: map[string]domain.Answer{"category": {Choice: "billing", Confidence: &c, Probabilities: map[string]float64{"billing": .60, "general": .25}}}}
	got := (Engine{Cfg: config.PolicyConfig{MinCategoryConfidence: .75, MinChoiceMargin: .15}}).Evaluate(d)
	if got.Action != "human_review" {
		t.Fatalf("got %s", got.Action)
	}
}
