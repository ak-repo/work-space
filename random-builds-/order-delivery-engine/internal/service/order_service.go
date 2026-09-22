package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/models"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
	"github.com/ak-repo/order-delivery-engine/internal/repository"
	"github.com/ak-repo/order-delivery-engine/internal/websocket"
	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo       *repository.OrderRepository
	driverRepo      *repository.DriverRepository
	assignment      *AssignmentService
	hub             *websocket.Hub
	defaultPriority int
	logger          *slog.Logger
	obsCfg          config.ObservabilityConfig
}

func NewOrderService(or *repository.OrderRepository, dr *repository.DriverRepository, as *AssignmentService, hub *websocket.Hub, cfg config.DefaultsConfig, logger *slog.Logger, obsCfg config.ObservabilityConfig) *OrderService {
	return &OrderService{
		orderRepo:       or,
		driverRepo:      dr,
		assignment:      as,
		hub:             hub,
		defaultPriority: cfg.OrderPriority,
		logger:          logger,
		obsCfg:          obsCfg,
	}
}

func (s *OrderService) CreateAndAssign(ctx context.Context, req *models.CreateOrderRequest) (*models.AssignmentResult, error) {
	if err := s.validateCreateOrder(req); err != nil {
		return nil, err
	}
	if req.Priority == 0 {
		req.Priority = s.defaultPriority
	}

	order := &models.Order{
		ID:              uuid.New().String(),
		CustomerName:    req.CustomerName,
		CustomerPhone:   req.CustomerPhone,
		PickupLat:       req.PickupLat,
		PickupLng:       req.PickupLng,
		PickupAddress:   req.PickupAddress,
		DeliveryLat:     req.DeliveryLat,
		DeliveryLng:     req.DeliveryLng,
		DeliveryAddress: req.DeliveryAddress,
		Status:          models.OrderPending,
		Priority:        req.Priority,
		Notes:           req.Notes,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("creating order: %w", err)
	}
	observability.LoggerFromContext(ctx, s.logger).InfoContext(ctx, "order created",
		slog.String("event", "order_created"),
		slog.String("order_id", order.ID),
	)
	observability.Audit(ctx, s.logger, s.obsCfg.AuditEnabled, "order.create", "order", order.ID, "success")
	s.hub.Broadcast(models.StatusUpdate{
		Type:    "order_update",
		OrderID: order.ID,
		Status:  string(order.Status),
		Message: fmt.Sprintf("Order %s created", order.ID),
		Data:    order,
	})

	result, err := s.assignment.AssignDriver(ctx, order)
	if err != nil {
		observability.LoggerFromContext(ctx, s.logger).WarnContext(ctx, "could not assign driver",
			slog.String("event", "order_assignment_failed"),
			slog.String("order_id", order.ID),
			slog.Any("error", err),
		)
		return &models.AssignmentResult{Order: *order}, nil
	}
	return result, nil
}

func (s *OrderService) validateCreateOrder(req *models.CreateOrderRequest) error {
	if strings.TrimSpace(req.CustomerName) == "" {
		return fmt.Errorf("customer_name is required")
	}
	if strings.TrimSpace(req.PickupAddress) == "" {
		return fmt.Errorf("pickup_address is required")
	}
	if strings.TrimSpace(req.DeliveryAddress) == "" {
		return fmt.Errorf("delivery_address is required")
	}
	if err := validateLatitude("pickup_lat", req.PickupLat); err != nil {
		return err
	}
	if err := validateLongitude("pickup_lng", req.PickupLng); err != nil {
		return err
	}
	if err := validateLatitude("delivery_lat", req.DeliveryLat); err != nil {
		return err
	}
	if err := validateLongitude("delivery_lng", req.DeliveryLng); err != nil {
		return err
	}
	if req.Priority != 0 && (req.Priority < 1 || req.Priority > 3) {
		return fmt.Errorf("priority must be between 1 and 3")
	}
	return nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, orderID string, status models.OrderStatus) (*models.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !isValidTransition(order.Status, status) {
		return nil, fmt.Errorf("invalid transition: %s → %s", order.Status, status)
	}
	tx, err := s.orderRepo.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	if err := s.orderRepo.UpdateStatusTx(ctx, tx, orderID, status); err != nil {
		tx.Rollback()
		return nil, err
	}
	if (status == models.OrderDelivered || status == models.OrderCancelled) && order.DriverID != nil {
		if err := s.driverRepo.ChangeActiveOrdersTx(ctx, tx, *order.DriverID, -1); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("updating driver active orders: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing status update: %w", err)
	}
	updated, _ := s.orderRepo.GetByID(ctx, orderID)
	driverID := ""
	if order.DriverID != nil {
		driverID = *order.DriverID
	}
	s.hub.Broadcast(models.StatusUpdate{
		Type:     "order_update",
		OrderID:  orderID,
		DriverID: driverID,
		Status:   string(status),
		Message:  fmt.Sprintf("Order %s status updated to %s", orderID, status),
		Data:     updated,
	})
	observability.Audit(ctx, s.logger, s.obsCfg.AuditEnabled, "order.update_status", "order", orderID, "success",
		slog.String("from_status", string(order.Status)),
		slog.String("to_status", string(status)),
	)
	if (status == models.OrderDelivered || status == models.OrderCancelled) && driverID != "" {
		driver, err := s.driverRepo.GetByID(ctx, driverID)
		if err == nil {
			s.hub.Broadcast(models.StatusUpdate{
				Type:     "driver_update",
				DriverID: driverID,
				Status:   string(driver.Status),
				Message:  fmt.Sprintf("Driver %s active orders updated", driver.Name),
				Data:     driver,
			})
		}
	}
	return updated, nil
}

func validateLatitude(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < -90 || value > 90 {
		return fmt.Errorf("%s must be between -90 and 90", name)
	}
	return nil
}

func validateLongitude(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < -180 || value > 180 {
		return fmt.Errorf("%s must be between -180 and 180", name)
	}
	return nil
}

func isValidTransition(from, to models.OrderStatus) bool {
	allowed := map[models.OrderStatus][]models.OrderStatus{
		models.OrderPending:   {models.OrderAssigned, models.OrderCancelled},
		models.OrderAssigned:  {models.OrderPickedUp, models.OrderCancelled},
		models.OrderPickedUp:  {models.OrderDelivered},
		models.OrderDelivered: {},
		models.OrderCancelled: {},
	}
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}
