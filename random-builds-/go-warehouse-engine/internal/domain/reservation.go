package domain

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusReleased  Status = "released"
)

type Reservation struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	OrderID   uuid.UUID `json:"order_id"`
	Quantity  int       `json:"quantity"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    Status    `json:"status"`
}
