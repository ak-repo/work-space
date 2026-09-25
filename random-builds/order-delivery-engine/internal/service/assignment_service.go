package service

import (
	"container/heap"
	"context"
	"fmt"
	"log/slog"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/models"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
	"github.com/ak-repo/order-delivery-engine/internal/repository"
	"github.com/ak-repo/order-delivery-engine/internal/websocket"

	"sync"
)

type AssignmentService struct {
	mu                sync.Mutex
	driverRepo        *repository.DriverRepository
	orderRepo         *repository.OrderRepository
	hub               *websocket.Hub
	maxSearchRadiusKm float64
	distanceWeight    float64
	capacityWeight    float64
	distanceSmoothing float64
	logger            *slog.Logger
	obsCfg            config.ObservabilityConfig
}

func NewAssignmentService(dr *repository.DriverRepository, or *repository.OrderRepository, hub *websocket.Hub, cfg config.AssignmentConfig, logger *slog.Logger, obsCfg config.ObservabilityConfig) *AssignmentService {
	return &AssignmentService{
		driverRepo:        dr,
		orderRepo:         or,
		hub:               hub,
		maxSearchRadiusKm: cfg.MaxSearchRadiusKm,
		distanceWeight:    cfg.DistanceWeight,
		capacityWeight:    cfg.CapacityWeight,
		distanceSmoothing: cfg.DistanceSmoothing,
		logger:            logger,
		obsCfg:            obsCfg,
	}
}

func (s *AssignmentService) AssignDriver(ctx context.Context, order *models.Order) (*models.AssignmentResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	drivers, err := s.driverRepo.GetAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching available drivers: %w", err)
	}
	if len(drivers) == 0 {
		return nil, fmt.Errorf("no available drivers at the moment")
	}

	pq := BuildDriverQueue(drivers, order.PickupLat, order.PickupLng, s.maxSearchRadiusKm, s.distanceWeight, s.capacityWeight, s.distanceSmoothing)
	if pq.Len() == 0 {
		return nil, fmt.Errorf("no drivers within %.0fkm radius", s.maxSearchRadiusKm)
	}

	best := heap.Pop(pq).(*models.DriverHeapItem)

	tx, err := s.orderRepo.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	if err := s.orderRepo.AssignDriverTx(ctx, tx, order.ID, best.Driver.ID); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("assigning driver: %w", err)
	}
	if err := s.driverRepo.ChangeActiveOrdersTx(ctx, tx, best.Driver.ID, +1); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("updating driver active orders: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing assignment: %w", err)
	}

	updated, err := s.orderRepo.GetByID(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	result := &models.AssignmentResult{
		Order:    *updated,
		Driver:   best.Driver,
		Distance: best.Driver.Distance,
		Score:    best.Score,
	}

	s.hub.Broadcast(models.StatusUpdate{
		Type:     "order_update",
		OrderID:  order.ID,
		DriverID: best.Driver.ID,
		Status:   string(models.OrderAssigned),
		Message:  fmt.Sprintf("Order assigned to driver %s (%.2f km away)", best.Driver.Name, best.Driver.Distance),
		Data:     result,
	})
	if driver, err := s.driverRepo.GetByID(ctx, best.Driver.ID); err == nil {
		s.hub.Broadcast(models.StatusUpdate{
			Type:     "driver_update",
			DriverID: best.Driver.ID,
			Status:   string(driver.Status),
			Message:  fmt.Sprintf("Driver %s active orders updated", driver.Name),
			Data:     driver,
		})
	}

	observability.LoggerFromContext(ctx, s.logger).InfoContext(ctx, "order assigned",
		slog.String("event", "order_assigned"),
		slog.String("order_id", order.ID),
		slog.String("driver_id", best.Driver.ID),
		slog.Float64("score", best.Score),
		slog.Float64("distance_km", best.Driver.Distance),
	)
	return result, nil
}

func (s *AssignmentService) RouteBatch(ctx context.Context, driverID string, orderIDs []string) (*models.BatchAssignmentResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if driverID == "" {
		return nil, fmt.Errorf("driver_id is required")
	}
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("order_ids is required")
	}

	driver, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return nil, err
	}
	if driver.Status != models.DriverAvailable {
		return nil, fmt.Errorf("driver must be available")
	}
	result := &models.BatchAssignmentResult{DriverID: driverID}

	assigned := 0
	for _, orderID := range orderIDs {
		order, err := s.orderRepo.GetByID(ctx, orderID)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "failed", Reason: err.Error()})
			continue
		}
		if order.Status != models.OrderPending {
			result.Skipped++
			result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "skipped", Reason: "order is not pending"})
			continue
		}
		if driver.ActiveOrders >= driver.Capacity {
			result.Skipped++
			result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "skipped", Reason: "driver is at capacity"})
			continue
		}
		tx, err := s.orderRepo.BeginTx(ctx, nil)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "failed", Reason: err.Error()})
			continue
		}
		if err := s.orderRepo.AssignDriverTx(ctx, tx, orderID, driverID); err != nil {
			tx.Rollback()
			result.Failed++
			result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "failed", Reason: err.Error()})
			continue
		}
		if err := s.driverRepo.ChangeActiveOrdersTx(ctx, tx, driverID, +1); err != nil {
			tx.Rollback()
			result.Failed++
			result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "failed", Reason: err.Error()})
			continue
		}
		if err := tx.Commit(); err != nil {
			result.Failed++
			result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "failed", Reason: err.Error()})
			continue
		}
		driver.ActiveOrders++
		assigned++
		result.Assigned++
		result.Results = append(result.Results, models.BatchAssignmentItem{OrderID: orderID, Status: "assigned"})

		s.hub.Broadcast(models.StatusUpdate{
			Type:     "order_update",
			OrderID:  orderID,
			DriverID: driverID,
			Status:   string(models.OrderAssigned),
			Message:  fmt.Sprintf("Batch assigned to driver %s", driver.Name),
		})
	}
	observability.LoggerFromContext(ctx, s.logger).InfoContext(ctx, "batch assignment completed",
		slog.String("event", "batch_assignment_completed"),
		slog.String("driver_id", driverID),
		slog.Int("assigned", assigned),
		slog.Int("failed", result.Failed),
		slog.Int("skipped", result.Skipped),
	)
	observability.Audit(ctx, s.logger, s.obsCfg.AuditEnabled, "order.batch_assign", "driver", driverID, "success",
		slog.Int("assigned", assigned),
		slog.Int("failed", result.Failed),
		slog.Int("skipped", result.Skipped),
	)
	if updatedDriver, err := s.driverRepo.GetByID(ctx, driverID); err == nil {
		s.hub.Broadcast(models.StatusUpdate{
			Type:     "driver_update",
			DriverID: driverID,
			Status:   string(updatedDriver.Status),
			Message:  fmt.Sprintf("Driver %s active orders updated", updatedDriver.Name),
			Data:     updatedDriver,
		})
	}
	return result, nil
}
