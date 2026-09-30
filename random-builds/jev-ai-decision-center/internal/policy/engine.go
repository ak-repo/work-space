package policy

import (
	"github.com/example/jev-ai-decision-center/internal/config"
	"github.com/example/jev-ai-decision-center/internal/domain"
)

type Engine struct{ Cfg config.PolicyConfig }

func (e Engine) Evaluate(d domain.DecisionResult) domain.PolicyOutcome {
	cat, ok := d.Answers["category"]
	if !ok || cat.Confidence == nil {
		return outcome("human_review", "missing_category_confidence", nil)
	}
	if *cat.Confidence < e.Cfg.MinCategoryConfidence {
		return outcome("human_review", "low_category_confidence", map[string]any{"confidence": *cat.Confidence})
	}
	if margin(cat.Probabilities) < e.Cfg.MinChoiceMargin {
		return outcome("human_review", "small_choice_margin", map[string]any{"margin": margin(cat.Probabilities)})
	}
	if cat.Choice == "security" {
		return outcome("security_review", "security_category", nil)
	}
	if esc, ok := d.Answers["needs_escalation"]; ok && esc.Probability != nil && *esc.Probability >= e.Cfg.EscalationProbability {
		return outcome("page_engineer", "high_escalation_probability", map[string]any{"probability": *esc.Probability})
	}
	if next, ok := d.Answers["next_action"]; ok && next.Choice == "observe" {
		return outcome("observe", "model_observe_with_policy_gate", nil)
	}
	return outcome("route_normal", "confident_standard_route", map[string]any{"category": cat.Choice})
}

func margin(m map[string]float64) float64 {
	var a, b float64
	for _, v := range m {
		if v > a {
			b = a
			a = v
		} else if v > b {
			b = v
		}
	}
	return a - b
}
func outcome(a, r string, d map[string]any) domain.PolicyOutcome {
	return domain.PolicyOutcome{Action: a, ReasonCode: r, Details: d}
}
