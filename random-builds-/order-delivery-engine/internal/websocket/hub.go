package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/models"
)

type Client struct {
	ID      string
	Channel chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
	logger  *slog.Logger
	obsCfg  config.ObservabilityConfig
}

func NewHub(logger *slog.Logger, obsCfg config.ObservabilityConfig) *Hub {
	return &Hub{clients: make(map[string]*Client), logger: logger, obsCfg: obsCfg}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID] = c
	h.logger.Debug("sse client registered", slog.String("client_id", c.ID), slog.Int("connected_clients", len(h.clients)))
}

func (h *Hub) Unregister(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.clients[clientID]; ok {
		close(c.Channel)
		delete(h.clients, clientID)
		h.logger.Debug("sse client unregistered", slog.String("client_id", clientID), slog.Int("connected_clients", len(h.clients)))
	}
}

func (h *Hub) Broadcast(update models.StatusUpdate) {
	data, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("failed to marshal broadcast", slog.Any("error", err), slog.String("event", "sse_broadcast_error"))
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.Channel <- data:
		default:
			if h.obsCfg.WarnOnDroppedSSEEvent {
				h.logger.Warn("sse client channel full, dropping event",
					slog.String("event", "sse_event_dropped"),
					slog.String("client_id", c.ID),
					slog.String("update_type", update.Type),
					slog.String("order_id", update.OrderID),
					slog.String("driver_id", update.DriverID),
				)
			}
		}
	}
}

func (h *Hub) Logger() *slog.Logger {
	if h.logger == nil {
		return slog.Default()
	}
	return h.logger
}
