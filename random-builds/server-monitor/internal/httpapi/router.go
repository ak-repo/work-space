package httpapi

import (
	"log"
	"net/http"
	"time"
)

func NewRouter(handler *Handler, dashboard http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", dashboard)
	mux.HandleFunc("GET /api/snapshot", handler.Snapshot)
	mux.HandleFunc("GET /api/system", handler.System)
	mux.HandleFunc("GET /api/database", handler.Database)
	mux.HandleFunc("GET /api/health", handler.Health)
	mux.HandleFunc("GET /health/live", handler.Live)
	mux.HandleFunc("GET /health/ready", handler.Ready)

	return loggingMiddleware(recoverMiddleware(mux))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered: %v", recovered)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
