package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/google/uuid"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"

	EventHTTPAccess      = "http_access"
	EventHTTPError       = "http_error"
	EventPanicRecovered  = "panic_recovered"
	EventDBQuery         = "db_query"
	EventSSEConnected    = "sse_connected"
	EventSSEDisconnected = "sse_disconnected"
	EventAudit           = "audit"
)

type Config = config.ObservabilityConfig

func NewLogger(cfg Config, environment string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	level := new(slog.LevelVar)
	level.Set(parseLevel(cfg.LogLevel))
	opts := &slog.HandlerOptions{Level: level, AddSource: cfg.AddSource}

	var handler slog.Handler
	if strings.EqualFold(cfg.LogFormat, "json") {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	attrs := []any{
		slog.String("service", valueOrDefault(cfg.ServiceName, "order-delivery-engine")),
		slog.String("environment", valueOrDefault(environment, "dev")),
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, slog.String("version", cfg.ServiceVersion))
	}
	return slog.New(handler).With(attrs...)
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func LoggerFromContext(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
		return logger
	}
	if fallback != nil {
		return fallback
	}
	return slog.Default()
}

func NewRequestID() string {
	return uuid.NewString()
}

func RequestIDFromHeader(r *http.Request, cfg Config) string {
	header := valueOrDefault(cfg.RequestIDHeader, "X-Request-ID")
	if cfg.TrustIncomingRequestID {
		if requestID := strings.TrimSpace(r.Header.Get(header)); requestID != "" {
			return requestID
		}
	}
	return NewRequestID()
}

func ClientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	return r.RemoteAddr
}

func LogHTTPError(ctx context.Context, fallback *slog.Logger, status int, msg string) {
	logger := LoggerFromContext(ctx, fallback)
	attrs := []any{
		slog.String("event", EventHTTPError),
		slog.Int("status", status),
		slog.String("error", msg),
	}
	if requestID := RequestID(ctx); requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}
	if status >= http.StatusInternalServerError {
		logger.ErrorContext(ctx, "http handler error", attrs...)
		return
	}
	logger.WarnContext(ctx, "http handler error", attrs...)
}

func Audit(ctx context.Context, fallback *slog.Logger, enabled bool, action, resourceType, resourceID, result string, attrs ...any) {
	if !enabled {
		return
	}
	logger := LoggerFromContext(ctx, fallback)
	base := []any{
		slog.String("event", EventAudit),
		slog.String("action", action),
		slog.String("resource_type", resourceType),
		slog.String("resource_id", resourceID),
		slog.String("result", result),
	}
	if requestID := RequestID(ctx); requestID != "" {
		base = append(base, slog.String("request_id", requestID))
	}
	base = append(base, attrs...)
	logger.InfoContext(ctx, "audit event", base...)
}

func Stack() string {
	return string(debug.Stack())
}

func DurationMillis(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

func Truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func NormalizeSQL(query string, max int) string {
	query = strings.Join(strings.Fields(query), " ")
	return Truncate(query, max)
}

func SQLOperation(query string) string {
	fields := strings.Fields(strings.TrimSpace(query))
	if len(fields) == 0 {
		return "unknown"
	}
	return strings.ToUpper(fields[0])
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func GinAccessLine(now time.Time, status int, duration time.Duration, clientIP, method, path, requestID string) string {
	base := fmt.Sprintf("[GIN] %s | %3d | %13s | %s | %-6s | %s", now.Format("2006/01/02 - 15:04:05"), status, duration, clientIP, method, path)
	if requestID == "" {
		return base
	}
	return base + " | req_id=" + requestID
}
