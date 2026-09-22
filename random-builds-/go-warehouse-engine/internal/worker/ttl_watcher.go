package worker

import (
	"context"
	"log"
	"time"
)

// ExpiryReleaser is implemented by InventoryService.
type ExpiryReleaser interface {
	ReleaseExpired(ctx context.Context)
}

// StartTTLWatcher runs a ticker loop that calls ReleaseExpired on every interval.
// It exits cleanly when ctx is cancelled (e.g. on SIGINT/SIGTERM).
func StartTTLWatcher(ctx context.Context, svc ExpiryReleaser, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Printf("[ttl-watcher] started (interval=%s)", interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("[ttl-watcher] stopped")
			return
		case <-ticker.C:
			svc.ReleaseExpired(ctx)
		}
	}
}
