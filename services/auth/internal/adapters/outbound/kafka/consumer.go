package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/auth/internal/core/app"
)

const (
	CustomerEventsTopic      = "customer.events"
	CustomerCreatedEventType = "customer.created"
)

// DedupeStore — интерфейс дедупликации событий по event_id.
type DedupeStore interface {
	MarkProcessed(ctx context.Context, eventID string) (bool, error)
}

// Consumer слушает customer.events и обновляет users.customer_id.
type Consumer struct {
	consumer    *kafkautil.Consumer
	groupID     string
	dedupeStore DedupeStore
	applyUC     *app.ApplyCustomerCreatedUseCase
	log         *slog.Logger
}

func NewConsumer(
	brokers []string,
	groupID string,
	dedupeStore DedupeStore,
	applyUC *app.ApplyCustomerCreatedUseCase,
	log *slog.Logger,
) *Consumer {
	return &Consumer{
		consumer:    kafkautil.NewConsumer(brokers),
		groupID:     groupID,
		dedupeStore: dedupeStore,
		applyUC:     applyUC,
		log:         log,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	return c.consumer.Subscribe(ctx, CustomerEventsTopic, c.groupID, c.handleEvent)
}

type customerCreatedPayload struct {
	UserID     string `json:"user_id"`
	CustomerID string `json:"customer_id"`
}

func (c *Consumer) handleEvent(ctx context.Context, envelope kafkautil.Envelope) error {
	alreadyProcessed, err := c.dedupeStore.MarkProcessed(ctx, envelope.EventID)
	if err != nil {
		return fmt.Errorf("dedupe check: %w", err)
	}
	if alreadyProcessed {
		c.log.Info("событие уже обработано, пропускаем", "event_id", envelope.EventID)
		return nil
	}

	if envelope.EventType != CustomerCreatedEventType {
		c.log.Warn("неизвестный тип события, пропускаем",
			"event_type", envelope.EventType, "event_id", envelope.EventID)
		return nil
	}

	var payload customerCreatedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal customer.created payload: %w", err)
	}

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}
	customerID, err := uuid.Parse(payload.CustomerID)
	if err != nil {
		return fmt.Errorf("invalid customer_id: %w", err)
	}

	if err := c.applyUC.ApplyCustomerCreated(ctx, app.ApplyCustomerCreatedInput{
		UserID:     userID,
		CustomerID: customerID,
	}); err != nil {
		return fmt.Errorf("apply customer.created: %w", err)
	}

	c.log.Info("customer_id привязан к пользователю",
		"user_id", userID, "customer_id", customerID)
	return nil
}
