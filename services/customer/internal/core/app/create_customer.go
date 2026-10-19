package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

const customerCreatedEventType = "customer.created"

type EventPublisher interface {
	Publish(ctx context.Context, topic string, envelope kafkautil.Envelope) error
}

type CreateCustomerUseCase struct {
	repo                domain.CustomerRepository
	eventPublisher      EventPublisher
	customerEventsTopic string
}

func NewCreateCustomerUseCase(
	repo domain.CustomerRepository,
	eventPublisher EventPublisher,
	customerEventsTopic string,
) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{
		repo:                repo,
		eventPublisher:      eventPublisher,
		customerEventsTopic: customerEventsTopic,
	}
}

type CreateCustomerInput struct {
	UserID uuid.UUID
}

type customerCreatedPayload struct {
	UserID     uuid.UUID `json:"user_id"`
	CustomerID uuid.UUID `json:"customer_id"`
}

func (uc *CreateCustomerUseCase) CreateCustomer(
	ctx context.Context,
	in CreateCustomerInput,
) (domain.Customer, error) {
	if in.UserID == uuid.Nil {
		return domain.Customer{}, domain.ErrInvalidCustomerID
	}

	customer := domain.Customer{
		UserID: in.UserID,
		Status: domain.StatusNew,
	}

	created, err := uc.repo.Create(ctx, customer)
	if err != nil {
		return domain.Customer{}, err
	}

	if err := uc.publishCustomerCreated(ctx, created); err != nil {
		slog.Error("не удалось опубликовать customer.created",
			"user_id", created.UserID,
			"customer_id", created.ID,
			"error", err,
		)
	}

	return created, nil
}

func (uc *CreateCustomerUseCase) publishCustomerCreated(
	ctx context.Context,
	customer domain.Customer,
) error {
	envelope, err := kafkautil.NewEnvelope(customerCreatedEventType, customerCreatedPayload{
		UserID:     customer.UserID,
		CustomerID: customer.ID,
	})
	if err != nil {
		return err
	}

	return uc.eventPublisher.Publish(ctx, uc.customerEventsTopic, envelope)
}
