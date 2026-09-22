package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
	"github.com/ak-repo/order-delivery-engine/internal/websocket"
	"github.com/google/uuid"
)

type SSEHandler struct {
	hub                 *websocket.Hub
	clientChannelBuffer int
	logger              *slog.Logger
	obsCfg              config.ObservabilityConfig
}

func NewSSEHandler(hub *websocket.Hub, cfg config.SSEConfig, logger *slog.Logger, obsCfg config.ObservabilityConfig) *SSEHandler {
	return &SSEHandler{
		hub:                 hub,
		clientChannelBuffer: cfg.ClientChannelBuffer,
		logger:              logger,
		obsCfg:              obsCfg,
	}
}

func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	client := &websocket.Client{
		ID:      uuid.New().String(),
		Channel: make(chan []byte, h.clientChannelBuffer),
	}
	started := time.Now()
	eventsSent := 0
	if h.obsCfg.LogSSEConnect {
		observability.LoggerFromContext(r.Context(), h.logger).InfoContext(r.Context(), "sse client connected",
			slog.String("event", observability.EventSSEConnected),
			slog.String("request_id", observability.RequestID(r.Context())),
			slog.String("client_id", client.ID),
			slog.String("client_ip", observability.ClientIP(r)),
		)
	}
	h.hub.Register(client)
	defer func() {
		h.hub.Unregister(client.ID)
		if h.obsCfg.LogSSEDisconnect {
			observability.LoggerFromContext(r.Context(), h.logger).InfoContext(r.Context(), "sse client disconnected",
				slog.String("event", observability.EventSSEDisconnected),
				slog.String("request_id", observability.RequestID(r.Context())),
				slog.String("client_id", client.ID),
				slog.Float64("duration_ms", observability.DurationMillis(time.Since(started))),
				slog.Int("events_sent", eventsSent),
			)
		}
	}()

	fmt.Fprintf(w, "event: connected\ndata: {\"client_id\":\"%s\"}\n\n", client.ID)
	flusher.Flush()
	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case msg, ok := <-client.Channel:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: update\ndata: %s\n\n", msg)
			flusher.Flush()
			eventsSent++
		case <-keepAlive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
