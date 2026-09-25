package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	requestsTotal atomic.Int64
	checksTotal   atomic.Int64
	checksFailed  atomic.Int64
	active        atomic.Int64

	mu           sync.RWMutex
	totalLatency time.Duration
	samples      int64
}

type Snapshot struct {
	RequestsTotal    int64 `json:"requests_total"`
	ChecksTotal      int64 `json:"checks_total"`
	ChecksFailed     int64 `json:"checks_failed"`
	ActiveRequests   int64 `json:"active_requests"`
	AverageLatencyMS int64 `json:"average_latency_ms"`
}

func (m *Metrics) BeginRequest() func() {
	m.requestsTotal.Add(1)
	m.active.Add(1)
	return func() { m.active.Add(-1) }
}

func (m *Metrics) RecordCheck(latency time.Duration, failed bool) {
	m.checksTotal.Add(1)
	if failed {
		m.checksFailed.Add(1)
	}
	m.mu.Lock()
	m.totalLatency += latency
	m.samples++
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() Snapshot {
	m.mu.RLock()
	average := int64(0)
	if m.samples > 0 {
		average = (m.totalLatency / time.Duration(m.samples)).Milliseconds()
	}
	m.mu.RUnlock()
	return Snapshot{
		RequestsTotal:    m.requestsTotal.Load(),
		ChecksTotal:      m.checksTotal.Load(),
		ChecksFailed:     m.checksFailed.Load(),
		ActiveRequests:   m.active.Load(),
		AverageLatencyMS: average,
	}
}
