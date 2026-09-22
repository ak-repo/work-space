package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ak-repo/order-delivery-engine/internal/models"
	"github.com/ak-repo/order-delivery-engine/internal/repository"
	"github.com/ak-repo/order-delivery-engine/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
	orderRepo    *repository.OrderRepository
	assignSvc    *service.AssignmentService
	logger       *slog.Logger
}

func NewOrderHandler(os *service.OrderService, or *repository.OrderRepository, as *service.AssignmentService, logger *slog.Logger) *OrderHandler {
	return &OrderHandler{orderService: os, orderRepo: or, assignSvc: as, logger: logger}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	result, err := h.orderService.CreateAndAssign(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "must be") {
			writeError(r.Context(), w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: result})
}

func (h *OrderHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	var orders []models.Order
	var err error
	if statusFilter != "" {
		orders, err = h.orderRepo.GetByStatus(r.Context(), models.OrderStatus(statusFilter))
	} else {
		orders, err = h.orderRepo.GetAll(r.Context())
	}
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: orders})
}

func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/api/orders/")
	order, err := h.orderRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: order})
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/orders/")
	id := strings.TrimSuffix(path, "/status")
	var req models.UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid JSON")
		return
	}
	order, err := h.orderService.UpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: order})
}

func (h *OrderHandler) BatchAssign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DriverID string   `json:"driver_id"`
		OrderIDs []string `json:"order_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid JSON")
		return
	}
	result, err := h.assignSvc.RouteBatch(r.Context(), body.DriverID, body.OrderIDs)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "must be") {
			status = http.StatusBadRequest
		}
		writeError(r.Context(), w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: result, Message: "batch assignment done"})
}
