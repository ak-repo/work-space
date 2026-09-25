package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/concurrent-health-checker/internal/checker"
	"example.com/concurrent-health-checker/internal/metrics"
)

const (
	MaxURLs        = 500
	MaxWorkers     = 50
	MinTimeout     = 50 * time.Millisecond
	MaxTimeout     = 30 * time.Second
	DefaultWorkers = 10
	DefaultTimeout = 3 * time.Second
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	checker *checker.Checker
	metrics *metrics.Metrics
}

type Response struct {
	Total      int              `json:"total"`
	Healthy    int              `json:"healthy"`
	Failed     int              `json:"failed"`
	DurationMS int64            `json:"duration_ms"`
	Results    []checker.Result `json:"results"`
}

func New(checker *checker.Checker, metrics *metrics.Metrics) *Service {
	return &Service{checker: checker, metrics: metrics}
}

func (s *Service) Check(ctx context.Context, urls []string, workers int, timeout time.Duration) (Response, error) {
	if len(urls) == 0 || len(urls) > MaxURLs {
		return Response{}, fmt.Errorf("%w: urls must contain between 1 and %d entries", ErrInvalidInput, MaxURLs)
	}
	if workers < 1 || workers > MaxWorkers {
		return Response{}, fmt.Errorf("%w: workers must be between 1 and %d", ErrInvalidInput, MaxWorkers)
	}
	if timeout < MinTimeout || timeout > MaxTimeout {
		return Response{}, fmt.Errorf("%w: timeout must be between %d and %d milliseconds", ErrInvalidInput, MinTimeout.Milliseconds(), MaxTimeout.Milliseconds())
	}

	finish := s.metrics.BeginRequest()
	defer finish()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	results, err := checker.RunPool(ctx, workers, urls, s.checker.Check)
	if err != nil {
		return Response{}, fmt.Errorf("run checks: %w", err)
	}
	response := Response{Total: len(results), Results: results, DurationMS: time.Since(started).Milliseconds()}
	for _, result := range results {
		if result.Healthy {
			response.Healthy++
		} else {
			response.Failed++
		}
		s.metrics.RecordCheck(time.Duration(result.LatencyMS)*time.Millisecond, !result.Healthy)
	}
	return response, nil
}
