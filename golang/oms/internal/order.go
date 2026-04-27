package internal

// DeliveryType represents the type of delivery
type DeliveryType string

const (
	DeliveryInstant   DeliveryType = "instant"
	DeliveryScheduled DeliveryType = "scheduled"
)

// Order status constants for instant delivery
const (
	StatusCreated        = "CREATED"
	StatusAccepted       = "ACCEPTED"   // instant: after creation
	StatusConfirmed      = "CONFIRMED"  // scheduled: after creation
	StatusPickedUp       = "PICKED_UP"  // instant
	StatusOnTheWay       = "ON_THE_WAY" // instant
	StatusPacked         = "PACKED"     // scheduled
	StatusShipped        = "SHIPPED"    // scheduled
	StatusOutForDelivery = "OUT_FOR_DELIVERY" // scheduled
	StatusDelivered      = "DELIVERED"
)

// Order is the core domain model
type Order struct {
	ID           string       `json:"id"`
	ProductID    int          `json:"product_id"`
	Quantity     int          `json:"quantity"`
	DeliveryType DeliveryType `json:"delivery_type"`
	Status       string       `json:"status"`
}

// CreateOrderRequest is the expected JSON body for POST /orders
type CreateOrderRequest struct {
	ProductID    int          `json:"product_id"`
	Quantity     int          `json:"quantity"`
	DeliveryType DeliveryType `json:"delivery_type"`
}

// UpdateStatusRequest is the expected JSON body for PUT /orders/{id}/status
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// ErrorResponse is a standard error envelope
type ErrorResponse struct {
	Error string `json:"error"`
}

// InstantWorkflow defines valid status transitions for instant delivery
var InstantWorkflow = []string{
	StatusCreated,
	StatusAccepted,
	StatusPickedUp,
	StatusOnTheWay,
	StatusDelivered,
}

// ScheduledWorkflow defines valid status transitions for scheduled delivery
var ScheduledWorkflow = []string{
	StatusCreated,
	StatusConfirmed,
	StatusPacked,
	StatusShipped,
	StatusOutForDelivery,
	StatusDelivered,
}
