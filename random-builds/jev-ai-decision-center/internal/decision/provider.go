package decision

import (
	"context"
	"github.com/example/jev-ai-decision-center/internal/domain"
)

type Question struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type Request struct {
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
	Model     string              `json:"model"`
}

type Provider interface {
	Decide(ctx context.Context, req Request) (domain.DecisionResult, error)
}

func IncidentQuestions() map[string]Question {
	return map[string]Question{
		"category": {Type: "choice", Instructions: "Classify the primary operational issue.", Criteria: map[string]string{
			"billing":        "Payment, invoice, subscription, refund, or charge issue.",
			"technical":      "Application bug, integration failure, or software malfunction.",
			"infrastructure": "Service, server, database, network, or availability issue.",
			"security":       "Authentication abuse, suspicious access, data exposure, or security event.",
			"general":        "Does not fit the other operational categories.",
		}},
		"severity": {Type: "score", Instructions: "Rate the operational severity.", Criteria: []string{
			"Low: little or no user impact", "Medium: limited degradation or individual user impact", "High: major degradation or many users affected", "Critical: outage, severe security risk, or major business impact",
		}},
		"needs_escalation": {Type: "noul", Instructions: "This event requires immediate human escalation."},
		"next_action": {Type: "choice", Instructions: "Choose the safest immediate handling action.", Criteria: map[string]string{
			"observe": "Record and monitor; no immediate intervention.", "route_support": "Send to normal support workflow.", "page_engineer": "Escalate to on-call engineer.", "security_review": "Send to security review workflow.", "human_review": "Needs a human decision.",
		}},
	}
}
