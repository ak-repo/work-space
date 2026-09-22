package worker

import (
	"context"
	"log"

	"warehouse-engine/internal/service"
)

// StartReorderWorker listens on the reorder alert channel.
// In production: replace log.Printf with an email / Slack / webhook call.
func StartReorderWorker(ctx context.Context, ch <-chan service.ReorderEvent) {
	log.Println("[reorder-worker] started")
	for {
		select {
		case <-ctx.Done():
			log.Println("[reorder-worker] stopped")
			return
		case event := <-ch:
			log.Printf("[reorder-worker] LOW STOCK ALERT sku=%s product_id=%s available=%d — trigger supplier reorder",
				event.SKU, event.ProductID, event.Available)
		}
	}
}
