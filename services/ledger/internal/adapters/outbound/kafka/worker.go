package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/Mazik-kun/mini-core/services/ledger/internal/adapters/outbound/postgres/db"
	"github.com/segmentio/kafka-go"
)

type OutboxWorker struct {
	q *db.Queries
	writer *kafka.Writer
	log *slog.Logger
}

func NewOutboxWorker(q *db.Queries, brokers []string, topic string, log *slog.Logger) *OutboxWorker{
	w := &kafka.Writer{
		Addr: kafka.TCP(brokers...),
		Topic: topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &OutboxWorker{
		q: q,
		writer: w,
		log: log,
	}
}

func (w *OutboxWorker) Run(ctx context.Context) error{
	w.log.Info("outbox worker started", "topic", w.writer.Topic)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer w.writer.Close()

	for {
		select{
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			events, err := w.q.GetPendingOutboxEvents(ctx)
			if err != nil{
				w.log.Error("outbox:failed to fetch events", "error", err)
				continue
			}
			for _, event := range events {
				err := w.writer.WriteMessages(ctx, kafka.Message{
					Key: []byte(event.ID.String()),
					Value: event.Payload,
				})
				if err != nil {
					w.log.Error("outbox: failed to send to kafka", "error", err)
					break
				}

				if err := w.q.MarkOutboxEventAsSent(ctx, event.ID); err != nil{
					w.log.Error("outbox: failed to mark as sent", "error", err)
				} else {
					w.log.Debug("outbox: event sent", "event_id:", event.ID, "type:", event.EventType)
				}
			}
		}
	}
}