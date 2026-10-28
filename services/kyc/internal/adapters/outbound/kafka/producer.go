package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
)

type Producer struct {
	p *kafkautil.Producer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{p: kafkautil.NewProducer(brokers)}
}

func (k *Producer) Publish(ctx context.Context, topic string, eventType string, payload interface{}) error {
	// 1. Сериализуем payload в JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 2. Формируем Envelope (как в customer/auth)
	env := kafkautil.Envelope{
		EventID:    uuid.Must(uuid.NewV7()).String(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC(),
		Payload:    data,
	}

	return k.p.Publish(ctx, topic, env)
}

func (k *Producer) Close() {
	k.p.Close()
}