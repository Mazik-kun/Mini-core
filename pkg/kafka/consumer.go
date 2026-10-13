package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	handlerMaxAttempts = 3
	handlerInitBackoff = 500 * time.Millisecond
	dlqTopicSuffix     = ".dlq"
)

// Consumer читает события из Kafka с группами.
type Consumer struct {
	brokers []string
}

// NewConsumer создаёт консьюмера для указанных брокеров.
func NewConsumer(brokers []string) *Consumer {
	return &Consumer{brokers: brokers}
}

// Subscribe читает топик в группе и передаёт конверты в handlerFunc.
// Ошибка handler повторяется до 3 раз; затем конверт уходит в <topic>.dlq,
// offset коммитится. Сообщения с битым JSON логируются и пропускаются.
func (consumer *Consumer) Subscribe(
	ctx context.Context,
	topic string,
	groupID string,
	handlerFunc func(ctx context.Context, envelope Envelope) error,
) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  consumer.brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10 << 20,
	})
	defer reader.Close()

	dlqTopic := topic + dlqTopicSuffix
	publisher := NewProducer(consumer.brokers)
	defer publisher.Close()

	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			slog.Error("не удалось прочитать сообщение", "topic", topic, "error", err)
			return err
		}

		var envelope Envelope
		if err := json.Unmarshal(message.Value, &envelope); err != nil {
			slog.Error("битый конверт, сообщение пропущено",
				"topic", message.Topic, "offset", message.Offset, "error", err)
			if commitErr := reader.CommitMessages(ctx, message); commitErr != nil {
				slog.Error("не удалось закоммитить offset", "error", commitErr)
			}
			continue
		}

		handlerErr := handleWithRetry(ctx, handlerFunc, envelope)
		if handlerErr != nil {
			slog.Error("событие не обработано после ретраев, отправка в DLQ",
				"event_id", envelope.EventID, "topic", topic, "error", handlerErr)
			if publishErr := publisher.Publish(ctx, dlqTopic, envelope); publishErr != nil {
				slog.Error("не удалось отправить в DLQ, offset не коммитим",
					"event_id", envelope.EventID, "error", publishErr)
				continue
			}
		}

		if err := reader.CommitMessages(ctx, message); err != nil {
			slog.Error("не удалось закоммитить offset", "error", err)
		}
	}
}

// handleWithRetry вызывает handlerFunc до handlerMaxAttempts раз с экспоненциальным backoff.
func handleWithRetry(
	ctx context.Context,
	handlerFunc func(ctx context.Context, envelope Envelope) error,
	envelope Envelope,
) error {
	var lastErr error
	backoff := handlerInitBackoff

	for attempt := 1; attempt <= handlerMaxAttempts; attempt++ {
		lastErr = handlerFunc(ctx, envelope)
		if lastErr == nil {
			return nil
		}

		slog.Warn("обработка события не удалась",
			"event_id", envelope.EventID,
			"attempt", attempt,
			"error", lastErr)

		if attempt == handlerMaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}

	return lastErr
}