package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/concurrent-health-checker/internal/checker"
	"example.com/concurrent-health-checker/internal/metrics"
	"example.com/concurrent-health-checker/internal/service"
)

type fakeService struct {
	response service.Response
	err      error
}

func (f fakeService) Check(context.Context, []string, int, time.Duration) (service.Response, error) {
	return f.response, f.err
}

func testRouter(svc checkService, stats *metrics.Metrics) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(svc, stats).Routes(logger)
}

func TestHandlerValidRequestAndEndpoints(t *testing.T) {
	t.Parallel()
	stats := &metrics.Metrics{}
	router := testRouter(fakeService{response: service.Response{Total: 1, Healthy: 1, Results: []checker.Result{{URL: "https://example.com", Healthy: true}}}}, stats)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/check", strings.NewReader(`{"urls":["https://example.com"],"workers":1,"timeout_ms":1000}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("X-Request-ID") == "" {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response checkResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || response.RequestID == "" || response.Total != 1 {
		t.Fatalf("unexpected response: %+v err=%v", response, err)
	}

	for _, path := range []string{"/health", "/metrics"} {
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, recorder.Code)
		}
	}
}

func TestHandlerRejectsInvalidRequests(t *testing.T) {
	t.Parallel()
	router := testRouter(fakeService{err: service.ErrInvalidInput}, &metrics.Metrics{})
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"urls":`},
		{name: "unknown field", body: `{"urls":["https://example.com"],"extra":true}`},
		{name: "multiple values", body: `{"urls":["https://example.com"]} {}`},
		{name: "invalid workers", body: `{"urls":["https://example.com"],"workers":-1,"timeout_ms":1000}`},
		{name: "invalid timeout", body: `{"urls":["https://example.com"],"workers":1,"timeout_ms":-1}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/check", strings.NewReader(test.body)))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestHandlerMapsTimeout(t *testing.T) {
	t.Parallel()
	router := testRouter(fakeService{err: context.DeadlineExceeded}, &metrics.Metrics{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/check", strings.NewReader(`{"urls":["https://example.com"]}`)))
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
