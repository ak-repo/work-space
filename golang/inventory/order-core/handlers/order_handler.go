package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"order-core/models"
	"order-core/services"
)

type OrderHandler struct {
	service *services.OrderService
}

func NewOrderHandler(service *services.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.handleCreateOrder)
	mux.HandleFunc("GET /orders/{id}", h.handleGetOrder)
	mux.HandleFunc("POST /orders/{id}/confirm", h.handleConfirmOrder)
	mux.HandleFunc("POST /orders/{id}/cancel", h.handleCancelOrder)
}

func (h *OrderHandler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.service.CreateOrder(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidOrderPayload):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, services.ErrStockUnavailable):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, models.CreateOrderResponse{OrderID: order.ID, Status: order.Status})
}

func (h *OrderHandler) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.PathValue("id"))
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "missing order id")
		return
	}

	order, err := h.service.GetOrder(orderID)
	if err != nil {
		if errors.Is(err, services.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) handleConfirmOrder(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.PathValue("id"))
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "missing order id")
		return
	}

	order, err := h.service.ConfirmOrder(r.Context(), orderID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, services.ErrInvalidOrderState):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(r.PathValue("id"))
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "missing order id")
		return
	}

	order, err := h.service.CancelOrder(r.Context(), orderID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, services.ErrInvalidOrderState):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
