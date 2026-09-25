package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"example.com/server-monitor/internal/monitor"
)

type Handler struct {
	monitor *monitor.Service
}

func NewHandler(service *monitor.Service) *Handler {
	return &Handler{monitor: service}
}

func (h *Handler) Snapshot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.monitor.Snapshot(r.Context()))
}

func (h *Handler) System(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.monitor.Snapshot(r.Context()).System)
}

func (h *Handler) Database(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.monitor.Snapshot(r.Context()).Database)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	snapshot := h.monitor.Snapshot(r.Context())
	status := http.StatusOK
	if snapshot.OverallStatus == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]any{
		"status":      snapshot.OverallStatus,
		"server_time": snapshot.ServerTime,
		"system":      snapshot.System.Status,
		"database":    snapshot.Database.Status,
	})
}

func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "alive",
		"time":   time.Now(),
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	snapshot := h.monitor.Snapshot(r.Context())
	ready := snapshot.System.Status != "degraded" && (!snapshot.Database.Configured || snapshot.Database.Status == "healthy")
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]any{
		"ready":       ready,
		"system":      snapshot.System.Status,
		"database":    snapshot.Database.Status,
		"server_time": snapshot.ServerTime,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
