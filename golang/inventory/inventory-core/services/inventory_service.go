package services

import (
	"errors"
	"fmt"
	"sync"

	"inventory-core/models"
)

var (
	ErrInvalidItem          = errors.New("invalid items payload")
	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrReservationNotFound  = errors.New("reservation not found")
	ErrReservationFinalized = errors.New("reservation already released")
)

type InventoryService struct {
	mu           sync.Mutex
	products     map[string]models.Product
	stock        map[string]int
	reservations map[string]*models.Reservation
	nextID       int
}

func NewInventoryService(products []models.Product, initialStock []models.Stock) *InventoryService {
	productMap := make(map[string]models.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	stockMap := make(map[string]int, len(initialStock))
	for _, s := range initialStock {
		stockMap[s.ProductID] = s.AvailableQty
	}

	return &InventoryService{
		products:     productMap,
		stock:        stockMap,
		reservations: make(map[string]*models.Reservation),
		nextID:       1,
	}
}

func (s *InventoryService) Check(items []models.Item) (models.CheckResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateItems(items); err != nil {
		return models.CheckResponse{}, err
	}

	resp := models.CheckResponse{AllAvailable: true, Items: make([]models.AvailabilityItem, 0, len(items))}
	for _, item := range items {
		available := s.stock[item.ProductID]
		canFulfill := available >= item.Quantity
		if !canFulfill {
			resp.AllAvailable = false
		}
		resp.Items = append(resp.Items, models.AvailabilityItem{
			ProductID:  item.ProductID,
			Requested:  item.Quantity,
			Available:  available,
			CanFulfill: canFulfill,
		})
	}

	return resp, nil
}

func (s *InventoryService) Reserve(orderID string, items []models.Item) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if orderID == "" {
		return "", ErrInvalidItem
	}
	if err := validateItems(items); err != nil {
		return "", err
	}

	for _, item := range items {
		if s.stock[item.ProductID] < item.Quantity {
			return "", ErrInsufficientStock
		}
	}

	for _, item := range items {
		s.stock[item.ProductID] -= item.Quantity
	}

	reservationID := fmt.Sprintf("res-%d", s.nextID)
	s.nextID++

	reservedItems := make([]models.Item, len(items))
	copy(reservedItems, items)

	s.reservations[reservationID] = &models.Reservation{
		ReservationID: reservationID,
		OrderID:       orderID,
		Items:         reservedItems,
		Status:        models.ReservationReserved,
	}

	return reservationID, nil
}

func (s *InventoryService) Confirm(reservationID string) (models.ReservationStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, ok := s.reservations[reservationID]
	if !ok {
		return "", ErrReservationNotFound
	}
	if res.Status == models.ReservationReleased {
		return "", ErrReservationFinalized
	}

	res.Status = models.ReservationConfirmed
	return res.Status, nil
}

func (s *InventoryService) Release(reservationID string) (models.ReservationStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, ok := s.reservations[reservationID]
	if !ok {
		return "", ErrReservationNotFound
	}
	if res.Status == models.ReservationReleased {
		return "", ErrReservationFinalized
	}

	for _, item := range res.Items {
		s.stock[item.ProductID] += item.Quantity
	}
	res.Status = models.ReservationReleased

	return res.Status, nil
}

func validateItems(items []models.Item) error {
	if len(items) == 0 {
		return ErrInvalidItem
	}
	for _, item := range items {
		if item.ProductID == "" || item.Quantity <= 0 {
			return ErrInvalidItem
		}
	}
	return nil
}
