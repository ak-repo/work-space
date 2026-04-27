package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"order-core/models"
)

var (
	ErrInvalidOrderPayload = errors.New("invalid order payload")
	ErrOrderNotFound       = errors.New("order not found")
	ErrInvalidOrderState   = errors.New("invalid order state transition")
)

type OrderService struct {
	mu              sync.Mutex
	orders          map[string]*models.Order
	nextID          int
	inventoryClient *InventoryClient
}

func NewOrderService(inventoryClient *InventoryClient) *OrderService {
	return &OrderService{
		orders:          make(map[string]*models.Order),
		nextID:          1,
		inventoryClient: inventoryClient,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req models.CreateOrderRequest) (models.Order, error) {
	if err := validateCreateRequest(req); err != nil {
		return models.Order{}, err
	}

	orderID := s.generateOrderID()

	if err := s.inventoryClient.Check(ctx, req.Items); err != nil {
		return models.Order{}, err
	}

	reservationID, err := s.inventoryClient.Reserve(ctx, orderID, req.Items)
	if err != nil {
		return models.Order{}, err
	}

	status := models.StatusAccepted
	if req.DeliveryType == models.DeliveryScheduled {
		status = models.StatusConfirmed
	}

	order := models.Order{
		ID:            orderID,
		UserID:        req.UserID,
		Items:         copyItems(req.Items),
		DeliveryType:  req.DeliveryType,
		Status:        status,
		ReservationID: reservationID,
	}

	s.mu.Lock()
	s.orders[order.ID] = &order
	s.mu.Unlock()

	return order, nil
}

func (s *OrderService) ConfirmOrder(ctx context.Context, orderID string) (models.Order, error) {
	s.mu.Lock()
	order, ok := s.orders[orderID]
	if !ok {
		s.mu.Unlock()
		return models.Order{}, ErrOrderNotFound
	}

	nextStatus, err := nextConfirmStatus(*order)
	if err != nil {
		s.mu.Unlock()
		return models.Order{}, err
	}
	reservationID := order.ReservationID
	s.mu.Unlock()

	if err := s.inventoryClient.Confirm(ctx, reservationID); err != nil {
		return models.Order{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	order.Status = nextStatus
	return *order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) (models.Order, error) {
	s.mu.Lock()
	order, ok := s.orders[orderID]
	if !ok {
		s.mu.Unlock()
		return models.Order{}, ErrOrderNotFound
	}
	if order.Status == models.StatusCancelled || order.Status == models.StatusDelivered {
		s.mu.Unlock()
		return models.Order{}, ErrInvalidOrderState
	}
	reservationID := order.ReservationID
	s.mu.Unlock()

	if err := s.inventoryClient.Release(ctx, reservationID); err != nil {
		return models.Order{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	order.Status = models.StatusCancelled
	return *order, nil
}

func (s *OrderService) GetOrder(orderID string) (models.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok {
		return models.Order{}, ErrOrderNotFound
	}
	return *order, nil
}

func (s *OrderService) generateOrderID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	orderID := fmt.Sprintf("ord-%d", s.nextID)
	s.nextID++
	return orderID
}

func validateCreateRequest(req models.CreateOrderRequest) error {
	if req.UserID == "" {
		return ErrInvalidOrderPayload
	}
	if req.DeliveryType != models.DeliveryInstant && req.DeliveryType != models.DeliveryScheduled {
		return ErrInvalidOrderPayload
	}
	if len(req.Items) == 0 {
		return ErrInvalidOrderPayload
	}
	for _, item := range req.Items {
		if item.ProductID == "" || item.Quantity <= 0 {
			return ErrInvalidOrderPayload
		}
	}
	return nil
}

func nextConfirmStatus(order models.Order) (models.OrderStatus, error) {
	switch order.DeliveryType {
	case models.DeliveryInstant:
		if order.Status != models.StatusAccepted {
			return "", ErrInvalidOrderState
		}
		return models.StatusPickedUp, nil
	case models.DeliveryScheduled:
		if order.Status != models.StatusConfirmed {
			return "", ErrInvalidOrderState
		}
		return models.StatusPacked, nil
	default:
		return "", ErrInvalidOrderPayload
	}
}

func copyItems(items []models.Item) []models.Item {
	cp := make([]models.Item, len(items))
	copy(cp, items)
	return cp
}
