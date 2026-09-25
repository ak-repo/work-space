package engine

import "log"

type Engine struct {
	rules []Rule
}

func NewEngine(rules []Rule) *Engine {
	return &Engine{rules: rules}

}

func (e *Engine) ApplyBestDiscount(order OrderInput) (AppliedRule, float64) {
	var best AppliedRule
	bestDiscount := 0.0
	bestPriority := -1

	for _, rule := range e.rules {
		if !ruleMatches(rule, order) {
			continue
		}

		discount := calculateDiscount(rule, order)

		if rule.Priority > bestPriority ||
			(rule.Priority == bestPriority && discount > bestDiscount) {

			bestPriority = rule.Priority
			bestDiscount = discount
			log.Println("rule:", rule.ID)
			best = AppliedRule{
				RuleId:         rule.ID,
				DiscountAmount: discount,
			}
		}
	}

	finalTotal := order.OrderTotal - bestDiscount
	if finalTotal < 0 {
		finalTotal = 0
	}

	return best, finalTotal
}
