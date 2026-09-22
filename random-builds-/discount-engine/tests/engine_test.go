package tests

import (
	"testing"

	"github.com/ak-repo/discount-engine/internal/engine"
)

func TestBestDiscountPriority(t *testing.T) {

	rules := []engine.Rule{
		{ID: "r1", DiscountPercentage: 10, Priority: 1},
		{ID: "r2", DiscountPercentage: 5, Priority: 2},
	}

	e := engine.NewEngine(rules)

	applied, _ := e.ApplyBestDiscount(engine.OrderInput{
		OrderTotal: 200,
	})

	if applied.RuleId != "r2" {
		t.Fatalf("expected r2, got %s", applied.RuleId)
	}

}
