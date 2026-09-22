package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/models"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
	"github.com/ak-repo/order-delivery-engine/internal/repository"
	"github.com/ak-repo/order-delivery-engine/internal/websocket"
	"github.com/google/uuid"
)

type DriverHandler struct {
	repo            *repository.DriverRepository
	orderRepo       *repository.OrderRepository
	hub             *websocket.Hub
	defaultCapacity int
	logger          *slog.Logger
	obsCfg          config.ObservabilityConfig
}

func NewDriverHandler(repo *repository.DriverRepository, orderRepo *repository.OrderRepository, hub *websocket.Hub, cfg config.DefaultsConfig, logger *slog.Logger, obsCfg config.ObservabilityConfig) *DriverHandler {
	return &DriverHandler{
		repo:            repo,
		orderRepo:       orderRepo,
		hub:             hub,
		defaultCapacity: cfg.DriverCapacity,
		logger:          logger,
		obsCfg:          obsCfg,
	}
}

func (h *DriverHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.validateCreateDriver(req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Capacity == 0 {
		req.Capacity = h.defaultCapacity
	}
	driver := &models.Driver{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Phone:        req.Phone,
		Status:       models.DriverAvailable,
		CurrentLat:   req.CurrentLat,
		CurrentLng:   req.CurrentLng,
		Capacity:     req.Capacity,
		ActiveOrders: 0,
	}
	if err := h.repo.Create(r.Context(), driver); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}
	observability.Audit(r.Context(), h.logger, h.obsCfg.AuditEnabled, "driver.create", "driver", driver.ID, "success")
	h.hub.Broadcast(models.StatusUpdate{
		Type:     "driver_update",
		DriverID: driver.ID,
		Status:   string(driver.Status),
		Message:  fmt.Sprintf("Driver %s created", driver.Name),
		Data:     driver,
	})
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: driver})
}

func (h *DriverHandler) validateCreateDriver(req models.CreateDriverRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Phone) == "" {
		return fmt.Errorf("phone is required")
	}
	if err := validateLatitude("current_lat", req.CurrentLat); err != nil {
		return err
	}
	if err := validateLongitude("current_lng", req.CurrentLng); err != nil {
		return err
	}
	if req.Capacity < 0 || req.Capacity > 10 {
		return fmt.Errorf("capacity must be between 1 and 10")
	}
	return nil
}

func (h *DriverHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	drivers, err := h.repo.GetAll(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: drivers})
}

func (h *DriverHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/api/drivers/")
	driver, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: driver})
}

func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/drivers/")
	id := strings.TrimSuffix(path, "/location")
	var req models.UpdateDriverLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := validateLatitude("lat", req.Lat); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLongitude("lng", req.Lng); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.UpdateLocation(r.Context(), id, req.Lat, req.Lng); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}
	observability.Audit(r.Context(), h.logger, h.obsCfg.AuditEnabled, "driver.update_location", "driver", id, "success")
	driver, err := h.repo.GetByID(r.Context(), id)
	if err == nil {
		h.hub.Broadcast(models.StatusUpdate{
			Type:     "driver_update",
			DriverID: id,
			Status:   string(driver.Status),
			Message:  fmt.Sprintf("Driver %s location updated", driver.Name),
			Data:     driver,
		})
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Message: "location updated"})
}

func (h *DriverHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/drivers/")
	id := strings.TrimSuffix(path, "/orders")
	orders, err := h.orderRepo.GetByDriver(r.Context(), id)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: orders})
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
