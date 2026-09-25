package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestMetricsConcurrentUpdates(t *testing.T) {
	t.Parallel()
	var stats Metrics
	finish := stats.BeginRequest()
	if got := stats.Snapshot().ActiveRequests; got != 1 {
		t.Fatalf("active requests = %d", got)
	}

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats.RecordCheck(10*time.Millisecond, i%2 == 0)
		}()
	}
	wg.Wait()
	finish()
	snapshot := stats.Snapshot()
	if snapshot.RequestsTotal != 1 || snapshot.ActiveRequests != 0 || snapshot.ChecksTotal != 100 || snapshot.ChecksFailed != 50 || snapshot.AverageLatencyMS != 10 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}
