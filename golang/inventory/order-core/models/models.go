package models

type Item struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type DeliveryType string

const (
	DeliveryInstant   DeliveryType = "instant"
	DeliveryScheduled DeliveryType = "scheduled"
)

type OrderStatus string

const (
	StatusCreated        OrderStatus = "CREATED"
	StatusAccepted       OrderStatus = "ACCEPTED"
	StatusConfirmed      OrderStatus = "CONFIRMED"
	StatusPickedUp       OrderStatus = "PICKED_UP"
	StatusOnTheWay       OrderStatus = "ON_THE_WAY"
	StatusPacked         OrderStatus = "PACKED"
	StatusShipped        OrderStatus = "SHIPPED"
	StatusOutForDelivery OrderStatus = "OUT_FOR_DELIVERY"
	StatusDelivered      OrderStatus = "DELIVERED"
	StatusCancelled      OrderStatus = "CANCELLED"
)

type Order struct {
	ID            string       `json:"id"`
	UserID        string       `json:"user_id"`
	Items         []Item       `json:"items"`
	DeliveryType  DeliveryType `json:"delivery_type"`
	Status        OrderStatus  `json:"status"`
	ReservationID string       `json:"reservation_id"`
}

type CreateOrderRequest struct {
	UserID       string       `json:"user_id"`
	Items        []Item       `json:"items"`
	DeliveryType DeliveryType `json:"delivery_type"`
}

type CreateOrderResponse struct {
	OrderID string      `json:"order_id"`
	Status  OrderStatus `json:"status"`
}
