package internal

import (
	"encoding/json"
	"net/http"
	"strings"
)

// OrderHandler wires HTTP routes to the OrderService
type OrderHandler struct {
	svc *OrderService
}

// New creates an OrderHandler backed by the given service
func NewHandler(svc *OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// -----------------------------------------------------------------
// POST /orders
// -----------------------------------------------------------------

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	order, err := h.svc.CreateOrder(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// -----------------------------------------------------------------
// GET /orders/{id}
// -----------------------------------------------------------------

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := extractSegment(r.URL.Path, "/orders/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	order, err := h.svc.GetOrder(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, order)
}

// -----------------------------------------------------------------
// PUT /orders/{id}/status
// -----------------------------------------------------------------

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Path: /orders/{id}/status  →  strip prefix and suffix
	id := extractIDFromStatusPath(r.URL.Path)
	if id == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	order, err := h.svc.UpdateStatus(id, req.Status)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "order not found" {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, order)
}

// -----------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, ErrorResponse{Error: msg})
}

// extractSegment strips a fixed prefix and returns the remainder
// e.g. "/orders/ORD-0001" with prefix "/orders/" → "ORD-0001"
func extractSegment(path, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	// Remove any trailing slashes or extra segments
	if idx := strings.Index(trimmed, "/"); idx != -1 {
		return ""
	}
	return trimmed
}

// extractIDFromStatusPath handles "/orders/{id}/status"
// Returns the {id} portion, or "" if the path doesn't match.
func extractIDFromStatusPath(path string) string {
	// Strip leading "/orders/"
	rest := strings.TrimPrefix(path, "/orders/")
	// rest should now be "{id}/status"
	parts := strings.Split(rest, "/")
	if len(parts) == 2 && parts[1] == "status" && parts[0] != "" {
		return parts[0]
	}
	return ""
}
