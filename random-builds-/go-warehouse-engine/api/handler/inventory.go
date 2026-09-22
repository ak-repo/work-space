package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"warehouse-engine/internal/domain"
	"warehouse-engine/internal/service"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
)

// Svc is the interface the handler depends on (makes it easy to mock in tests).
type Svc interface {
	Reserve(ctx context.Context, productID, orderID uuid.UUID, qty int) (domain.Reservation, error)
	Confirm(ctx context.Context, reservationID uuid.UUID) error
	Cancel(ctx context.Context, reservationID uuid.UUID) error
	ListProducts(ctx context.Context) ([]domain.Product, error)
	ListReservations(ctx context.Context) ([]domain.Reservation, error)
}

type InventoryHandler struct{ svc Svc }

func New(svc Svc) *InventoryHandler { return &InventoryHandler{svc: svc} }

// RegisterRoutes wires all API routes onto the provided ServeMux.
// Using Go 1.22 method+path pattern syntax: "METHOD /path"
func (h *InventoryHandler) RegisterRoutes(r *chi.Mux) {
	r.Use(middleware.Logger)

	r.Get("/health", h.health)
	r.Post("/reserve", h.reserve)
	r.Post("/confirm", h.confirm)
	r.Post("/cancel", h.cancel)
	r.Get("/api/products", h.listProducts)
	r.Get("/api/reservations", h.listReservations)
}

// ---------- request / response structs ----------

type reserveReq struct {
	ProductID string `json:"product_id"`
	OrderID   string `json:"order_id"`
	Quantity  int    `json:"quantity"`
}

type reserveResp struct {
	ID        uuid.UUID     `json:"id"`
	ProductID uuid.UUID     `json:"product_id"`
	OrderID   uuid.UUID     `json:"order_id"`
	Quantity  int           `json:"quantity"`
	ExpiresAt time.Time     `json:"expires_at"`
	Status    domain.Status `json:"status"`
}

type actionReq struct {
	ReservationID string `json:"reservation_id"`
}

// ---------- handlers ----------

func (h *InventoryHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *InventoryHandler) reserve(w http.ResponseWriter, r *http.Request) {
	var req reserveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid product_id")
		return
	}
	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid order_id")
		return
	}
	res, err := h.svc.Reserve(r.Context(), productID, orderID, req.Quantity)
	if err != nil {
		code := http.StatusConflict
		if errors.Is(err, service.ErrBadQuantity) {
			code = http.StatusBadRequest
		}
		writeErr(w, code, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reserveResp{
		ID:        res.ID,
		ProductID: res.ProductID,
		OrderID:   res.OrderID,
		Quantity:  res.Quantity,
		ExpiresAt: res.ExpiresAt,
		Status:    res.Status,
	})
}

func (h *InventoryHandler) confirm(w http.ResponseWriter, r *http.Request) {
	var req actionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	id, err := uuid.Parse(req.ReservationID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid reservation_id")
		return
	}
	if err := h.svc.Confirm(r.Context(), id); err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "confirmed"})
}

func (h *InventoryHandler) cancel(w http.ResponseWriter, r *http.Request) {
	var req actionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	id, err := uuid.Parse(req.ReservationID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid reservation_id")
		return
	}
	if err := h.svc.Cancel(r.Context(), id); err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cancelled"})
}

func (h *InventoryHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.ListProducts(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list products: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *InventoryHandler) listReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := h.svc.ListReservations(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list reservations: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reservations)
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
