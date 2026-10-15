package kafka

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/app"
)

const (
	AuthEventsTopic         = "auth.events"
	UserRegisteredEventType = "user_registered"
)

// UserRegisteredPayload — payload события auth.user_registered.
// Формат должен совпадать с auth/app/register.go:userRegisteredPayload.
type UserRegisteredPayload struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
}

// Consumer читает auth.events и создаёт профиль по событию user_registered.
type Consumer struct {
	brokers        []string
	groupID        string
	dedupeStore    kafkautil.DedupeStore
	createCustomer *app.CreateCustomerUseCase
	log            *slog.Logger
}

func NewConsumer(
	brokers []string,
	groupID string,
	dedupeStore kafkautil.DedupeStore,
	createCustomer *app.CreateCustomerUseCase,
	log *slog.Logger,
) *Consumer {
	return &Consumer{
		brokers:        brokers,
		groupID:        groupID,
		dedupeStore:    dedupeStore,
		createCustomer: createCustomer,
		log:            log,
	}
}

// Run блокируется, пока не отменят ctx. Retry и DLQ делает pkg/kafka.Subscribe.
func (c *Consumer) Run(ctx context.Context) error {
	subscriber := kafkautil.NewConsumer(c.brokers)
	return subscriber.Subscribe(ctx, AuthEventsTopic, c.groupID, c.handleEnvelope)
}

func (c *Consumer) handleEnvelope(ctx context.Context, envelope kafkautil.Envelope) error {
	if envelope.EventType != UserRegisteredEventType {
		return nil // чужое событие — пропускаем молча
	}

	var payload UserRegisteredPayload
	if err := envelope.UnmarshalPayload(&payload); err != nil {
		return err
	}

	if payload.Role != roles.RoleClient {
		return nil // офицеров и админов профиль не создаём
	}

	alreadyProcessed, err := c.dedupeStore.MarkProcessed(ctx, envelope.EventID)
	if err != nil {
		return err
	}
	if alreadyProcessed {
		c.log.Info("событие уже обработано, пропускаем", "event_id", envelope.EventID)
		return nil
	}

	_, err = c.createCustomer.CreateCustomer(ctx, app.CreateCustomerInput{UserID: payload.UserID})
	return err
}