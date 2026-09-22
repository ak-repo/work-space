package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ak-repo/go-trade-engine/internal/models"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
			// Optimization: Batching for high throughput
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) PublishOrder(order models.Order) error {
	msg, _ := json.Marshal(models.OrderMessage{Action: "new", Order: order})
	return p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(order.Pair), // Ensure partition ordering by Pair
			Value: msg,
		},
	)
}

func (p *Producer) PublishTrades(trades []models.Trade) {
	msgs := make([]kafka.Message, len(trades))
	for i, t := range trades {
		val, _ := json.Marshal(t)
		msgs[i] = kafka.Message{
			Key:   []byte(t.Pair),
			Value: val,
		}
	}
	if len(msgs) > 0 {
		p.writer.WriteMessages(context.Background(), msgs...)
	}
}
