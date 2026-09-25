package checker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckerStatusesAndValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		status    int
		healthy   bool
		errorKind ErrorKind
	}{
		{name: "healthy", status: http.StatusOK, healthy: true},
		{name: "server error", status: http.StatusInternalServerError, errorKind: ErrorHTTP},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
			}))
			defer server.Close()
			result := New(server.Client()).Check(context.Background(), server.URL)
			if result.Healthy != test.healthy || result.StatusCode != test.status || result.ErrorKind != test.errorKind {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}

	result := New(http.DefaultClient).Check(context.Background(), "file:///etc/passwd")
	if result.ErrorKind != ErrorInvalidURL || result.Healthy {
		t.Fatalf("unexpected invalid URL result: %+v", result)
	}
}

func TestCheckerTimeoutAndCancellation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	check := New(server.Client())

	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		result := check.Check(ctx, server.URL)
		if result.ErrorKind != ErrorTimeout {
			t.Fatalf("expected timeout, got %+v", result)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := check.Check(ctx, server.URL)
		if result.ErrorKind != ErrorCanceled || !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("expected cancellation, got %+v", result)
		}
	})
}

func TestCheckerNetworkFailure(t *testing.T) {
	t.Parallel()
	result := New(&http.Client{Timeout: time.Second}).Check(context.Background(), "http://127.0.0.1:1")
	if result.ErrorKind != ErrorNetwork {
		t.Fatalf("expected network error, got %+v", result)
	}
}
