package router

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
)

func requestIDMiddleware(next http.Handler, logger *slog.Logger, cfg config.ObservabilityConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := observability.RequestIDFromHeader(r, cfg)
		requestLogger := logger.With(
			slog.String("request_id", requestID),
			slog.String("client_ip", observability.ClientIP(r)),
			slog.String("method", r.Method),
			slog.String("path", observability.Truncate(r.URL.Path, cfg.MaxPathLength)),
		)

		w.Header().Set(valueOrDefault(cfg.RequestIDHeader, "X-Request-ID"), requestID)
		ctx := observability.WithRequestID(r.Context(), requestID)
		ctx = observability.WithLogger(ctx, requestLogger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func recoveryMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx := r.Context()
				observability.LoggerFromContext(ctx, logger).ErrorContext(ctx, "panic recovered",
					slog.String("event", observability.EventPanicRecovered),
					slog.Any("panic", recovered),
					slog.String("stack", observability.Stack()),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler, logger *slog.Logger, cfg config.ObservabilityConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		ctx := r.Context()
		path := r.URL.Path
		if cfg.LogQueryString && r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		path = observability.Truncate(path, cfg.MaxPathLength)
		attrs := []any{
			slog.String("event", observability.EventHTTPAccess),
			slog.String("request_id", observability.RequestID(ctx)),
			slog.String("method", r.Method),
			slog.String("path", path),
			slog.Int("status", rw.status),
			slog.Float64("duration_ms", observability.DurationMillis(duration)),
			slog.String("client_ip", observability.ClientIP(r)),
			slog.Int("response_bytes", rw.bytes),
		}
		if cfg.LogUserAgent {
			attrs = append(attrs, slog.String("user_agent", observability.Truncate(r.UserAgent(), 256)))
		}
		if rw.status >= http.StatusInternalServerError {
			if cfg.SlowRequestThreshold > 0 && duration >= cfg.SlowRequestThreshold && r.URL.Path != "/events" {
				attrs = append(attrs, slog.Bool("slow", true))
			}
			observability.LoggerFromContext(ctx, logger).ErrorContext(ctx, "http request failed", attrs...)
			return
		}
		if cfg.SlowRequestThreshold > 0 && duration >= cfg.SlowRequestThreshold && r.URL.Path != "/events" {
			attrs = append(attrs, slog.Bool("slow", true))
			observability.LoggerFromContext(ctx, logger).WarnContext(ctx, "slow http request", attrs...)
			return
		}
		if cfg.GinStyleAccess && !strings.EqualFold(cfg.LogFormat, "json") {
			observability.LoggerFromContext(ctx, logger).InfoContext(ctx, observability.GinAccessLine(time.Now(), rw.status, duration, observability.ClientIP(r), r.Method, path, observability.RequestID(ctx)), attrs...)
			return
		}
		observability.LoggerFromContext(ctx, logger).InfoContext(ctx, "http request", attrs...)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (rw *responseWriter) WriteHeader(status int) {
	if rw.wrote {
		return
	}
	rw.wrote = true
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wrote {
		rw.WriteHeader(rw.status)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
