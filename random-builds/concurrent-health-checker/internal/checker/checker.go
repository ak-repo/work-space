package checker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type ErrorKind string

const (
	ErrorInvalidURL ErrorKind = "invalid_url"
	ErrorTimeout    ErrorKind = "timeout"
	ErrorCanceled   ErrorKind = "canceled"
	ErrorNetwork    ErrorKind = "network"
	ErrorHTTP       ErrorKind = "http_failure"
)

var ErrInvalidURL = errors.New("invalid URL")

type Result struct {
	URL        string    `json:"url"`
	StatusCode int       `json:"status_code,omitempty"`
	LatencyMS  int64     `json:"latency_ms"`
	Healthy    bool      `json:"healthy"`
	Error      string    `json:"error,omitempty"`
	ErrorKind  ErrorKind `json:"error_kind,omitempty"`
}

type Checker struct {
	client *http.Client
}

func New(client *http.Client) *Checker {
	return &Checker{client: client}
}

func (c *Checker) Check(ctx context.Context, rawURL string) Result {
	result := Result{URL: rawURL}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		result.Error = fmt.Errorf("%w: only absolute HTTP(S) URLs are supported", ErrInvalidURL).Error()
		result.ErrorKind = ErrorInvalidURL
		return result
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		result.Error = fmt.Errorf("create request: %w", err).Error()
		result.ErrorKind = ErrorInvalidURL
		return result
	}

	started := time.Now()
	response, err := c.client.Do(request)
	result.LatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		result.Error = fmt.Errorf("check %s: %w", rawURL, err).Error()
		var networkError net.Error
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			result.ErrorKind = ErrorTimeout
		case errors.Is(err, context.Canceled):
			result.ErrorKind = ErrorCanceled
		case errors.As(err, &networkError) && networkError.Timeout():
			result.ErrorKind = ErrorTimeout
		default:
			result.ErrorKind = ErrorNetwork
		}
		return result
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))

	result.StatusCode = response.StatusCode
	result.Healthy = response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest
	if !result.Healthy {
		result.Error = fmt.Sprintf("HTTP status %d", response.StatusCode)
		result.ErrorKind = ErrorHTTP
	}
	return result
}
