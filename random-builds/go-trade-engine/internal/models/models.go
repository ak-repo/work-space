package models

import "encoding/json"

type OrderType string
type Side string

const (
	Limit  OrderType = "LIMIT"
	Market OrderType = "MARKET"
	Buy    Side      = "BUY"
	Sell   Side      = "SELL"
)

type Order struct {
	ID        string    `json:"id"`
	Pair      string    `json:"pair"`
	Side      Side      `json:"side"`
	Type      OrderType `json:"type"`
	Price     float64   `json:"price"`
	Amount    float64   `json:"amount"`
	Timestamp int64     `json:"timestamp"`
}

type Trade struct {
	MakerOrderID string  `json:"maker_order_id"`
	TakerOrderID string  `json:"taker_order_id"`
	Price        float64 `json:"price"`
	Amount       float64 `json:"amount"`
	Pair         string  `json:"pair"`
	Timestamp    int64   `json:"timestamp"`
}

// Kafka Messages
type OrderMessage struct {
	Action string `json:"action"` // "new", "cancel"
	Order  Order  `json:"order"`
}

func (o *Order) ToJSON() []byte {
	b, _ := json.Marshal(o)
	return b
}