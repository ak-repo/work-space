package models

import "time"

// DriverStatus represents availability of a driver
type DriverStatus string

const (
	DriverAvailable   DriverStatus = "available"
	DriverBusy        DriverStatus = "busy"
	DriverUnavailable DriverStatus = "unavailable"
)

// OrderStatus represents lifecycle state of an order
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderAssigned  OrderStatus = "assigned"
	OrderPickedUp  OrderStatus = "picked_up"
	OrderDelivered OrderStatus = "delivered"
	OrderCancelled OrderStatus = "cancelled"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Driver struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Phone        string       `json:"phone"`
	Status       DriverStatus `json:"status"`
	CurrentLat   float64      `json:"current_lat"`
	CurrentLng   float64      `json:"current_lng"`
	Capacity     int          `json:"capacity"`      // max orders at once
	ActiveOrders int          `json:"active_orders"` // current assigned count
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`

	// computed during assignment
	Score    float64 `json:"score,omitempty"`
	Distance float64 `json:"distance_km,omitempty"`
}

type Order struct {
	ID              string      `json:"id"`
	CustomerName    string      `json:"customer_name"`
	CustomerPhone   string      `json:"customer_phone"`
	PickupLat       float64     `json:"pickup_lat"`
	PickupLng       float64     `json:"pickup_lng"`
	PickupAddress   string      `json:"pickup_address"`
	DeliveryLat     float64     `json:"delivery_lat"`
	DeliveryLng     float64     `json:"delivery_lng"`
	DeliveryAddress string      `json:"delivery_address"`
	Status          OrderStatus `json:"status"`
	DriverID        *string     `json:"driver_id,omitempty"`
	Priority        int         `json:"priority"` // 1=normal, 2=express, 3=premium
	Notes           string      `json:"notes"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	AssignedAt      *time.Time  `json:"assigned_at,omitempty"`
	DeliveredAt     *time.Time  `json:"delivered_at,omitempty"`
}

// AssignmentResult is what gets returned after driver assignment
type AssignmentResult struct {
	Order    Order   `json:"order"`
	Driver   Driver  `json:"driver"`
	Distance float64 `json:"distance_km"`
	Score    float64 `json:"score"`
}

type BatchAssignmentItem struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
}

type BatchAssignmentResult struct {
	DriverID string                `json:"driver_id"`
	Results  []BatchAssignmentItem `json:"results"`
	Assigned int                   `json:"assigned"`
	Skipped  int                   `json:"skipped"`
	Failed   int                   `json:"failed"`
}

// StatusUpdate is pushed over WebSocket/SSE
type StatusUpdate struct {
	Type     string      `json:"type"` // "order_update" | "driver_update"
	OrderID  string      `json:"order_id,omitempty"`
	DriverID string      `json:"driver_id,omitempty"`
	Status   string      `json:"status"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data,omitempty"`
}

// DriverHeapItem is used in the priority queue for driver scoring
type DriverHeapItem struct {
	Driver Driver
	Score  float64 // higher = better
	Index  int
}

// CreateOrderRequest is the incoming JSON body for POST /orders
type CreateOrderRequest struct {
	CustomerName    string  `json:"customer_name"`
	CustomerPhone   string  `json:"customer_phone"`
	PickupLat       float64 `json:"pickup_lat"`
	PickupLng       float64 `json:"pickup_lng"`
	PickupAddress   string  `json:"pickup_address"`
	DeliveryLat     float64 `json:"delivery_lat"`
	DeliveryLng     float64 `json:"delivery_lng"`
	DeliveryAddress string  `json:"delivery_address"`
	Priority        int     `json:"priority"`
	Notes           string  `json:"notes"`
}

// CreateDriverRequest is the incoming JSON body for POST /drivers
type CreateDriverRequest struct {
	Name       string  `json:"name"`
	Phone      string  `json:"phone"`
	CurrentLat float64 `json:"current_lat"`
	CurrentLng float64 `json:"current_lng"`
	Capacity   int     `json:"capacity"`
}

// UpdateDriverLocationRequest updates driver GPS
type UpdateDriverLocationRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// UpdateOrderStatusRequest manually advances order status
type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}
