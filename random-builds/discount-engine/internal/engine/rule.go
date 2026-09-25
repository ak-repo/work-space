package engine

type Condition struct {
	MinOrderValue float64 `json:"min_order_value,omitempty"`
	CustomerType  string  `json:"customer_type,omitempty"`
}

type Rule struct {
	ID                 string    `json:"id"`
	Description        string    `json:"description"`
	Condition          Condition `json:"condition"`
	DiscountPercentage float64   `json:"discount_percentage,omitempty"`
	DiscountFixed      float64   `json:"discount_fixed,omitempty"`
	Priority           int       `json:"priority"`
}

type Response struct {
	AppliedRule string  `json:"applied_rule"`
	Discount    float64 `json:"discount"`
	FinalTotal  float64 `json:"final_total"`
}
