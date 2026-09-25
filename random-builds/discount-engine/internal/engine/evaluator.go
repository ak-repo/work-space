package engine

type OrderInput struct {
	OrderTotal   float64 `json:"order_total"`
	CustomerType string  `json:"customer_type"`
}

type AppliedRule struct {
	RuleId         string
	DiscountAmount float64
}

func ruleMatches(rule Rule, order OrderInput) bool {
	if rule.Condition.MinOrderValue > 0 &&
		order.OrderTotal < rule.Condition.MinOrderValue {
		return false
	}

	if rule.Condition.CustomerType != "" &&
		order.CustomerType != rule.Condition.CustomerType {
		return false
	}

	return true
}

func calculateDiscount(rule Rule, order OrderInput) float64 {
	if rule.DiscountPercentage > 0 {
		return order.OrderTotal * rule.DiscountPercentage / 100
	}
	return rule.DiscountFixed
}
