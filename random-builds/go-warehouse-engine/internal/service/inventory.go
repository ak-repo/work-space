package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"warehouse-engine/internal/domain"
	"warehouse-engine/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrBadQuantity       = errors.New("quantity must be >= 1")
)

// ReorderEvent is pushed onto the reorder channel when available stock
// drops at or below a product's reorder_at threshold.
type ReorderEvent struct {
	ProductID uuid.UUID
	SKU       string
	Available int
}

type InventoryService struct {
	products     *repository.ProductRepo
	reservations *repository.ReservationRepo
	ttl          time.Duration
	reorderCh    chan ReorderEvent
}

func NewInventoryService(
	products *repository.ProductRepo,
	reservations *repository.ReservationRepo,
	ttl time.Duration,
) *InventoryService {
	return &InventoryService{
		products:     products,
		reservations: reservations,
		ttl:          ttl,
		reorderCh:    make(chan ReorderEvent, 256),
	}
}

// ReorderCh exposes a read-only view of the reorder alert channel.
func (s *InventoryService) ReorderCh() <-chan ReorderEvent { return s.reorderCh }

// Reserve holds qty units of a product for a given order.
//
// Concurrency strategy — optimistic locking:
//  1. Read product (gets current version).
//  2. Check available stock.
//  3. UPDATE with version guard in WHERE clause.
//  4. If 0 rows updated another goroutine/request won the race → retry.
//  5. Up to 5 retries; each attempt re-reads the latest state.
func (s *InventoryService) Reserve(ctx context.Context, productID, orderID uuid.UUID, qty int) (domain.Reservation, error) {
	if qty < 1 {
		return domain.Reservation{}, ErrBadQuantity
	}

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		product, err := s.products.GetByID(ctx, productID)
		if err != nil {
			return domain.Reservation{}, fmt.Errorf("get product: %w", err)
		}
		if product.Available() < qty {
			return domain.Reservation{}, ErrInsufficientStock
		}

		if err := s.products.ReserveStock(ctx, product, qty); err != nil {
			if errors.Is(err, repository.ErrVersionConflict) {
				lastErr = err
				continue // another request got there first, retry
			}
			return domain.Reservation{}, fmt.Errorf("reserve stock: %w", err)
		}

		// Stock successfully held — now persist the reservation record.
		res := domain.Reservation{
			ID:        uuid.New(),
			ProductID: productID,
			OrderID:   orderID,
			Quantity:  qty,
			ExpiresAt: time.Now().Add(s.ttl),
			Status:    domain.StatusPending,
		}
		if err := s.reservations.Create(ctx, res); err != nil {
			// compensating action: undo the reserve to keep stock consistent
			_ = s.products.ReleaseStock(ctx, productID, qty)
			return domain.Reservation{}, fmt.Errorf("create reservation: %w", err)
		}

		s.maybeReorder(ctx, productID)
		return res, nil
	}
	return domain.Reservation{}, fmt.Errorf("reserve: all 5 retries exhausted: %w", lastErr)
}

// Confirm converts a pending reservation into a confirmed sale.
// Permanently deducts from total_stock.
func (s *InventoryService) Confirm(ctx context.Context, reservationID uuid.UUID) error {
	res, err := s.reservations.GetByID(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("get reservation: %w", err)
	}
	if res.Status != domain.StatusPending {
		return fmt.Errorf("reservation status is '%s', expected 'pending'", res.Status)
	}
	if time.Now().After(res.ExpiresAt) {
		return errors.New("reservation has expired")
	}
	if err := s.products.ConfirmStock(ctx, res.ProductID, res.Quantity); err != nil {
		return fmt.Errorf("confirm stock: %w", err)
	}
	return s.reservations.UpdateStatus(ctx, reservationID, domain.StatusConfirmed)
}

// Cancel releases the held stock back to available.
func (s *InventoryService) Cancel(ctx context.Context, reservationID uuid.UUID) error {
	res, err := s.reservations.GetByID(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("get reservation: %w", err)
	}
	if res.Status != domain.StatusPending {
		return fmt.Errorf("reservation status is '%s', expected 'pending'", res.Status)
	}
	if err := s.products.ReleaseStock(ctx, res.ProductID, res.Quantity); err != nil {
		return fmt.Errorf("release stock: %w", err)
	}
	return s.reservations.UpdateStatus(ctx, reservationID, domain.StatusReleased)
}

// ReleaseExpired is called by the TTL watcher goroutine.
// It finds all pending reservations past their expires_at and releases their stock.
func (s *InventoryService) ReleaseExpired(ctx context.Context) {
	expired, err := s.reservations.ListExpiredPending(ctx, time.Now())
	if err != nil {
		log.Printf("[ttl-watcher] list expired reservations: %v", err)
		return
	}
	for _, res := range expired {
		if err := s.products.ReleaseStock(ctx, res.ProductID, res.Quantity); err != nil {
			log.Printf("[ttl-watcher] release stock for reservation %s: %v", res.ID, err)
			continue
		}
		if err := s.reservations.UpdateStatus(ctx, res.ID, domain.StatusReleased); err != nil {
			log.Printf("[ttl-watcher] update status for reservation %s: %v", res.ID, err)
			continue
		}
		log.Printf("[ttl-watcher] released expired reservation %s (order=%s qty=%d)", res.ID, res.OrderID, res.Quantity)
	}
}

// ListProducts returns all products in the database.
func (s *InventoryService) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.products.ListAll(ctx)
}

// ListReservations returns all reservations in the database.
func (s *InventoryService) ListReservations(ctx context.Context) ([]domain.Reservation, error) {
	return s.reservations.ListAll(ctx)
}

// maybeReorder fires a reorder event if available stock has hit the threshold.
func (s *InventoryService) maybeReorder(ctx context.Context, productID uuid.UUID) {
	p, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return
	}
	if p.Available() <= p.ReorderAt {
		select {
		case s.reorderCh <- ReorderEvent{ProductID: p.ID, SKU: p.SKU, Available: p.Available()}:
		default:
			log.Printf("[reorder] channel full, dropped alert for SKU %s", p.SKU)
		}
	}
}
