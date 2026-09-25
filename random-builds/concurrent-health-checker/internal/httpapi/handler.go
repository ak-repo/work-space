package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"example.com/concurrent-health-checker/internal/metrics"
	"example.com/concurrent-health-checker/internal/service"
)

const maxBodyBytes = 64 << 10

type checkService interface {
	Check(context.Context, []string, int, time.Duration) (service.Response, error)
}

type Handler struct {
	service checkService
	metrics *metrics.Metrics
}

type checkRequest struct {
	URLs      []string `json:"urls"`
	Workers   int      `json:"workers"`
	TimeoutMS int64    `json:"timeout_ms"`
}

type checkResponse struct {
	RequestID string `json:"request_id"`
	service.Response
}

func NewHandler(service checkService, metrics *metrics.Metrics) *Handler {
	return &Handler{service: service, metrics: metrics}
}

func (h *Handler) Routes(logger MiddlewareLogger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /metrics", h.getMetrics)
	mux.HandleFunc("POST /api/v1/check", h.check)
	return Middleware(logger)(mux)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) getMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.metrics.Snapshot())
}

func (h *Handler) check(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input checkRequest
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := ensureEOF(decoder); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.Workers == 0 {
		input.Workers = service.DefaultWorkers
	}
	if input.TimeoutMS == 0 {
		input.TimeoutMS = service.DefaultTimeout.Milliseconds()
	}

	result, err := h.service.Check(r.Context(), input.URLs, input.Workers, time.Duration(input.TimeoutMS)*time.Millisecond)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, context.DeadlineExceeded):
			writeError(w, http.StatusGatewayTimeout, "health check timed out")
		case errors.Is(err, context.Canceled):
			writeError(w, http.StatusRequestTimeout, "health check canceled")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, checkResponse{RequestID: RequestID(r.Context()), Response: result})
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON value")
		}
		return fmt.Errorf("invalid trailing data: %w", err)
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
