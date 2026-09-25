package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/concurrent-health-checker/internal/checker"
	"example.com/concurrent-health-checker/internal/metrics"
)

func TestServiceAggregatesResults(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	stats := &metrics.Metrics{}
	svc := New(checker.New(server.Client()), stats)
	response, err := svc.Check(context.Background(), []string{server.URL, server.URL + "/bad"}, 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if response.Total != 2 || response.Healthy != 1 || response.Failed != 1 || stats.Snapshot().ChecksTotal != 2 {
		t.Fatalf("unexpected response: %+v metrics=%+v", response, stats.Snapshot())
	}
}

func TestServiceValidationAndTimeout(t *testing.T) {
	t.Parallel()
	svc := New(checker.New(http.DefaultClient), &metrics.Metrics{})
	if _, err := svc.Check(context.Background(), nil, 1, time.Second); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if _, err := svc.Check(context.Background(), []string{"https://example.com"}, MaxWorkers+1, time.Second); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid workers, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	timed := New(checker.New(server.Client()), &metrics.Metrics{})
	_, err := timed.Check(context.Background(), []string{server.URL}, 1, MinTimeout)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected timeout, got %v", err)
	}
}
