package checker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPoolPreservesOrderAndLimitsConcurrency(t *testing.T) {
	t.Parallel()
	inputs := []int{5, 4, 3, 2, 1, 0}
	var active atomic.Int64
	var maximum atomic.Int64
	results, err := RunPool(context.Background(), 3, inputs, func(_ context.Context, value int) int {
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(time.Duration(value) * time.Millisecond)
		active.Add(-1)
		return value * 2
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, result := range results {
		if result != inputs[i]*2 {
			t.Fatalf("result %d = %d", i, result)
		}
	}
	if maximum.Load() > 3 {
		t.Fatalf("concurrency exceeded worker count: %d", maximum.Load())
	}
}

func TestRunPoolCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := RunPool(ctx, 2, make([]int, 100), func(ctx context.Context, value int) int {
		<-ctx.Done()
		return value
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
}

func TestRunPoolHandlesFailedJobsWithoutDeadlock(t *testing.T) {
	t.Parallel()
	results, err := RunPool(context.Background(), 4, []int{1, 2, 3}, func(_ context.Context, value int) error {
		return errors.New("job failed")
	})
	if err != nil || len(results) != 3 {
		t.Fatalf("results=%d err=%v", len(results), err)
	}
}
