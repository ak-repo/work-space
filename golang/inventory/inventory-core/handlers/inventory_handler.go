package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"inventory-core/models"
	"inventory-core/services"
)

type InventoryHandler struct {
	service *services.InventoryService
}

func NewInventoryHandler(service *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{service: service}
}

func (h *InventoryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /inventory/check", h.handleCheck)
	mux.HandleFunc("POST /inventory/reserve", h.handleReserve)
	mux.HandleFunc("POST /inventory/confirm", h.handleConfirm)
	mux.HandleFunc("POST /inventory/release", h.handleRelease)
}

func (h *InventoryHandler) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req models.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.Check(req.Items)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *InventoryHandler) handleReserve(w http.ResponseWriter, r *http.Request) {
	var req models.ReserveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	reservationID, err := h.service.Reserve(req.OrderID, req.Items)
	if err != nil {
		if errors.Is(err, services.ErrInsufficientStock) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ReserveResponse{ReservationID: reservationID})
}

func (h *InventoryHandler) handleConfirm(w http.ResponseWriter, r *http.Request) {
	var req models.ReservationActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status, err := h.service.Confirm(req.ReservationID)
	if err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, services.ErrReservationNotFound) {
			statusCode = http.StatusNotFound
		}
		writeError(w, statusCode, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ReservationActionResponse{
		ReservationID: req.ReservationID,
		Status:        status,
	})
}

func (h *InventoryHandler) handleRelease(w http.ResponseWriter, r *http.Request) {
	var req models.ReservationActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status, err := h.service.Release(req.ReservationID)
	if err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, services.ErrReservationNotFound) {
			statusCode = http.StatusNotFound
		}
		writeError(w, statusCode, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ReservationActionResponse{
		ReservationID: req.ReservationID,
		Status:        status,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
