package models

type Product struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Stock struct {
	ProductID    string `json:"product_id"`
	AvailableQty int    `json:"available_qty"`
}

type ReservationStatus string

const (
	ReservationReserved  ReservationStatus = "RESERVED"
	ReservationConfirmed ReservationStatus = "CONFIRMED"
	ReservationReleased  ReservationStatus = "RELEASED"
)

type Item struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Reservation struct {
	ReservationID string            `json:"reservation_id"`
	OrderID       string            `json:"order_id"`
	Items         []Item            `json:"items"`
	Status        ReservationStatus `json:"status"`
}

type CheckRequest struct {
	Items []Item `json:"items"`
}

type AvailabilityItem struct {
	ProductID  string `json:"product_id"`
	Requested  int    `json:"requested"`
	Available  int    `json:"available"`
	CanFulfill bool   `json:"can_fulfill"`
}

type CheckResponse struct {
	AllAvailable bool               `json:"all_available"`
	Items        []AvailabilityItem `json:"items"`
}

type ReserveRequest struct {
	OrderID string `json:"order_id"`
	Items   []Item `json:"items"`
}

type ReserveResponse struct {
	ReservationID string `json:"reservation_id"`
}

type ReservationActionRequest struct {
	ReservationID string `json:"reservation_id"`
}

type ReservationActionResponse struct {
	ReservationID string            `json:"reservation_id"`
	Status        ReservationStatus `json:"status"`
}
