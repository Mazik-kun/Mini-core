package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer публикует события в Kafka.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer создаёт продюсера для указанных брокеров.
func NewProducer(brokers []string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		WriteTimeout: 10 * time.Second,
	}
	return &Producer{writer: writer}
}

// Publish отправляет конверт в топик.
func (producer *Producer) Publish(ctx context.Context, topic string, envelope Envelope) error {
	value, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Topic: topic,
		Key:   []byte(envelope.EventID),
		Value: value,
		Time:  envelope.OccurredAt,
	}
	return producer.writer.WriteMessages(ctx, message)
}

// Close закрывает writer.
func (producer *Producer) Close() error {
	return producer.writer.Close()
}