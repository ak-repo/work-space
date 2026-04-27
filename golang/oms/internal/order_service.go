package internal

import (
	"errors"
)

// OrderService holds business logic for orders
type OrderService struct {
	store *Store
}

// New creates an OrderService backed by the given store
func NewService(s *Store) *OrderService {
	return &OrderService{store: s}
}

// CreateOrder validates the request, applies initial status, persists, and returns the order
func (svc *OrderService) CreateOrder(req CreateOrderRequest) (*Order, error) {
	if req.ProductID <= 0 {
		return nil, errors.New("product_id must be a positive integer")
	}
	if req.Quantity <= 0 {
		return nil, errors.New("quantity must be a positive integer")
	}
	if req.DeliveryType != DeliveryInstant && req.DeliveryType != DeliveryScheduled {
		return nil, errors.New("delivery_type must be 'instant' or 'scheduled'")
	}

	// Determine initial status based on delivery type
	initialStatus := StatusCreated
	if req.DeliveryType == DeliveryInstant {
		initialStatus = StatusAccepted
	} else {
		initialStatus = StatusConfirmed
	}

	order := &Order{
		ProductID:    req.ProductID,
		Quantity:     req.Quantity,
		DeliveryType: req.DeliveryType,
		Status:       initialStatus,
	}

	svc.store.Save(order)
	return order, nil
}

// GetOrder retrieves an order by ID
func (svc *OrderService) GetOrder(id string) (*Order, error) {
	order, ok := svc.store.Get(id)
	if !ok {
		return nil, errors.New("order not found")
	}
	return order, nil
}

// UpdateStatus validates the new status against the order's workflow, then updates it
func (svc *OrderService) UpdateStatus(id, newStatus string) (*Order, error) {
	order, ok := svc.store.Get(id)
	if !ok {
		return nil, errors.New("order not found")
	}

	if newStatus == "" {
		return nil, errors.New("status field is required")
	}

	// Validate the new status belongs to the correct workflow
	if !isValidStatus(order.DeliveryType, newStatus) {
		return nil, errors.New("invalid status for delivery type '" + string(order.DeliveryType) + "'")
	}

	// Validate forward-only progression
	if !isForwardTransition(order.DeliveryType, order.Status, newStatus) {
		return nil, errors.New("status transition from '" + order.Status + "' to '" + newStatus + "' is not allowed")
	}

	svc.store.UpdateStatus(id, newStatus)
	order.Status = newStatus
	return order, nil
}

// isValidStatus checks whether a status string exists in the given delivery type's workflow
func isValidStatus(dt DeliveryType, status string) bool {
	workflow := workflowFor(dt)
	for _, s := range workflow {
		if s == status {
			return true
		}
	}
	return false
}

// isForwardTransition ensures the new status comes after the current one in the workflow
func isForwardTransition(dt DeliveryType, current, next string) bool {
	workflow := workflowFor(dt)
	currentIdx, nextIdx := -1, -1
	for i, s := range workflow {
		if s == current {
			currentIdx = i
		}
		if s == next {
			nextIdx = i
		}
	}
	return nextIdx > currentIdx
}

// workflowFor returns the ordered status slice for a delivery type
func workflowFor(dt DeliveryType) []string {
	if dt == DeliveryInstant {
		return InstantWorkflow
	}
	return ScheduledWorkflow
}
