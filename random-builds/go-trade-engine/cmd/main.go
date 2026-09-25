package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ak-repo/go-trade-engine/internal/engine"
	kafka_pkg "github.com/ak-repo/go-trade-engine/internal/kafka"
	"github.com/ak-repo/go-trade-engine/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

const (
	KafkaBroker = "localhost:9092"
	TopicOrders = "orders"
	TopicTrades = "trades"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	producer := kafka_pkg.NewProducer([]string{KafkaBroker}, TopicTrades)

	//Initialize OrderBook (In-Memory)
	ob := engine.NewOrderBook("BTC/USDT")

	go func() {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{KafkaBroker},
			Topic:    TopicOrders,
			GroupID:  "engine-group",
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		})

		log.Println("Engine started, listening for orders...")

		for {
			m, err := r.ReadMessage(context.Background())
			if err != nil {
				break
			}

			var payload models.OrderMessage
			json.Unmarshal(m.Value, &payload)

			trades, _ := ob.ProcessLimitOrder(&payload.Order)

			
			if len(trades) > 0 {
				go producer.PublishTrades(trades)
				go updateCandles(rdb, trades)
			}
		}
	}()

	r := gin.Default()

	orderWriter := &kafka.Writer{
		Addr:  kafka.TCP(KafkaBroker),
		Topic: TopicOrders,
	}

	r.POST("/order", func(c *gin.Context) {
		var order models.Order
		if err := c.BindJSON(&order); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		order.Timestamp = time.Now().UnixNano()

		// Push to Kafka
		msg, _ := json.Marshal(models.OrderMessage{Action: "new", Order: order})
		orderWriter.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(order.Pair),
			Value: msg,
		})

		c.JSON(202, gin.H{"status": "queued", "id": order.ID})
	})

	r.Run(":8080")
}

// Simple OHLCV Aggregation
func updateCandles(rdb *redis.Client, trades []models.Trade) {
	// In production, use Lua scripts for atomicity
	ctx := context.Background()
	for _, t := range trades {
		// Key: "candles:BTC/USDT:1m:<timestamp>"
		bucket := t.Timestamp / 1e9 / 60 * 60 // 1 minute bucket
		key := fmt.Sprintf("candle:%s:1m:%d", t.Pair, bucket)

		// Update Redis (simplified)
		// Set Open if not exists, Update High/Low/Close/Vol
		rdb.HSetNX(ctx, key, "open", t.Price)
		rdb.HIncrByFloat(ctx, key, "volume", t.Amount)
		rdb.HSet(ctx, key, "close", t.Price)
		// High/Low logic requires GET-CHECK-SET or Lua
	}
}
