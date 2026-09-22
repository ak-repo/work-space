package domain

import "github.com/google/uuid"

type Product struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SKU        string    `json:"sku"`
	TotalStock int       `json:"total_stock"`
	Reserved   int       `json:"reserved"`
	Version    int       `json:"version"`    // bumped on every mutation — used for optimistic locking
	ReorderAt  int       `json:"reorder_at"` // alert threshold: trigger reorder when Available() <= ReorderAt
}

// Available returns stock that is not currently held by any pending reservation.
func (p Product) Available() int {
	return p.TotalStock - p.Reserved
}
