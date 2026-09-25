package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ak-repo/order-delivery-engine/internal/models"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(ctx context.Context, w http.ResponseWriter, status int, msg string) {
	observability.LogHTTPError(ctx, slog.Default(), status, msg)
	writeJSON(w, status, models.APIResponse{Success: false, Error: msg})
}

func extractID(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}
