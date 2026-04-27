package internal

import (
	"net/http"
	"strings"
)

// RegisterRoutes wires all API routes to the default ServeMux.
// Route dispatch is done manually to avoid external dependencies.
func RegisterRoutes(mux *http.ServeMux, h *OrderHandler) {
	// POST /orders
	mux.HandleFunc("/orders", h.CreateOrder)

	// GET /orders/{id}   and   PUT /orders/{id}/status
	// Both share the "/orders/" prefix — we dispatch by path shape.
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// PUT /orders/{id}/status
		if strings.HasSuffix(path, "/status") && r.Method == http.MethodPut {
			h.UpdateStatus(w, r)
			return
		}

		// GET /orders/{id}
		if r.Method == http.MethodGet {
			h.GetOrder(w, r)
			return
		}

		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})
}
