package router

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/handlers"
)

func New(dh *handlers.DriverHandler, oh *handlers.OrderHandler, sh *handlers.SSEHandler, logger *slog.Logger, cfg config.ObservabilityConfig) http.Handler {
	mux := http.NewServeMux()

	// Serve static files for frontend
	mux.Handle("/", http.FileServer(http.Dir("public")))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/events", sh.Stream)

	// Drivers
	mux.HandleFunc("/api/drivers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dh.GetAll(w, r)
		case http.MethodPost:
			dh.Create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/drivers/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/location") && r.Method == http.MethodPut:
			dh.UpdateLocation(w, r)
		case strings.HasSuffix(path, "/orders") && r.Method == http.MethodGet:
			dh.GetOrders(w, r)
		case r.Method == http.MethodGet:
			dh.GetByID(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Orders
	mux.HandleFunc("/api/orders/batch-assign", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		oh.BatchAssign(w, r)
	})

	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			oh.GetAll(w, r)
		case http.MethodPost:
			oh.Create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/status") && r.Method == http.MethodPatch:
			oh.UpdateStatus(w, r)
		case r.Method == http.MethodGet:
			oh.GetByID(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return requestIDMiddleware(loggingMiddleware(recoveryMiddleware(corsMiddleware(mux), logger), logger, cfg), logger, cfg)
}
