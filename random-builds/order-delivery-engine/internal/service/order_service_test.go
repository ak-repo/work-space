package service

import (
	"testing"

	"github.com/ak-repo/order-delivery-engine/internal/models"
)

func TestValidateCreateOrder(t *testing.T) {
	svc := &OrderService{}

	valid := models.CreateOrderRequest{
		CustomerName:    "Customer",
		PickupAddress:   "Pickup",
		DeliveryAddress: "Delivery",
		PickupLat:       0,
		PickupLng:       0,
		DeliveryLat:     8.5,
		DeliveryLng:     76.9,
		Priority:        2,
	}

	tests := []struct {
		name    string
		mutate  func(*models.CreateOrderRequest)
		wantErr bool
	}{
		{name: "valid zero coordinates", wantErr: false},
		{name: "missing customer", mutate: func(req *models.CreateOrderRequest) { req.CustomerName = " " }, wantErr: true},
		{name: "missing pickup address", mutate: func(req *models.CreateOrderRequest) { req.PickupAddress = "" }, wantErr: true},
		{name: "missing delivery address", mutate: func(req *models.CreateOrderRequest) { req.DeliveryAddress = "" }, wantErr: true},
		{name: "pickup latitude out of range", mutate: func(req *models.CreateOrderRequest) { req.PickupLat = 91 }, wantErr: true},
		{name: "delivery longitude out of range", mutate: func(req *models.CreateOrderRequest) { req.DeliveryLng = -181 }, wantErr: true},
		{name: "priority out of range", mutate: func(req *models.CreateOrderRequest) { req.Priority = 4 }, wantErr: true},
		{name: "priority omitted", mutate: func(req *models.CreateOrderRequest) { req.Priority = 0 }, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			if tt.mutate != nil {
				tt.mutate(&req)
			}
			err := svc.validateCreateOrder(&req)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name string
		from models.OrderStatus
		to   models.OrderStatus
		want bool
	}{
		{name: "pending to assigned", from: models.OrderPending, to: models.OrderAssigned, want: true},
		{name: "assigned to picked up", from: models.OrderAssigned, to: models.OrderPickedUp, want: true},
		{name: "picked up to delivered", from: models.OrderPickedUp, to: models.OrderDelivered, want: true},
		{name: "pending to delivered blocked", from: models.OrderPending, to: models.OrderDelivered, want: false},
		{name: "delivered to picked up blocked", from: models.OrderDelivered, to: models.OrderPickedUp, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("isValidTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
