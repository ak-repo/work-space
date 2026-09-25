package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type MiddlewareLogger interface {
	Log(context.Context, slog.Level, string, ...any)
}

type contextKey string

const requestIDKey contextKey = "request_id"

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func Middleware(logger MiddlewareLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			requestID := newRequestID()
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)
			recorder := &responseRecorder{ResponseWriter: w}
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Log(ctx, slog.LevelError, "panic recovered", "request_id", requestID, "panic", recovered, "stack", string(debug.Stack()))
					if recorder.status == 0 {
						writeError(recorder, http.StatusInternalServerError, "internal server error")
					}
				}
				status := recorder.status
				if status == 0 {
					status = http.StatusOK
				}
				logger.Log(ctx, slog.LevelInfo, "request completed", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", status, "duration", time.Since(started))
			}()
			next.ServeHTTP(recorder, r.WithContext(ctx))
		})
	}
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func newRequestID() string {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(value[:])
}
