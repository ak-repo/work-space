package tests

import (
	"log"
	"testing"

	"github.com/ak-repo/go-trade-engine/internal/engine"
	"github.com/ak-repo/go-trade-engine/internal/models"
)

// go test -bench=. -cpu=1

func BenchmarkEngine(b *testing.B) {
	ob := engine.NewOrderBook("BTC/USDT")
	// Pre-fill orders
	for i := 0; i < b.N; i++ {
		log.Println("hii")

		ob.ProcessLimitOrder(&models.Order{
			Side: models.Buy, Price: 100.0, Amount: 1.0,
		})
	}

}
