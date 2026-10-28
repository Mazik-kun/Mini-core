package kafka

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/app"
)

const (
	KycSubmittedEventType = "kyc.application_submitted"
	KycApprovedEventType  = "kyc.application_approved"
	KycRejectedEventType  = "kyc.application_rejected"
)

type KycPayload struct {
	CustomerID string `json:"customer_id"`
	Reason     string `json:"reason,omitempty"`
}

type KycConsumer struct {
	brokers     []string
	groupID     string
	topic       string
	dedupeStore kafkautil.DedupeStore
	applyKyc    *app.ApplyKycDecisionUseCase
	log         *slog.Logger
}

func NewKycConsumer(
	brokers []string,
	groupID string,
	topic string,
	dedupeStore kafkautil.DedupeStore,
	applyKyc *app.ApplyKycDecisionUseCase,
	log *slog.Logger,
) *KycConsumer {
	return &KycConsumer{
		brokers:     brokers,
		groupID:     groupID,
		topic:       topic,
		dedupeStore: dedupeStore,
		applyKyc:    applyKyc,
		log:         log,
	}
}

func (c *KycConsumer) Run(ctx context.Context) error {
	subscriber := kafkautil.NewConsumer(c.brokers)
	return subscriber.Subscribe(ctx, c.topic, c.groupID, c.handleEnvelope)
}

func (c *KycConsumer) handleEnvelope(ctx context.Context, envelope kafkautil.Envelope) error {
	if envelope.EventType != KycSubmittedEventType &&
		envelope.EventType != KycApprovedEventType &&
		envelope.EventType != KycRejectedEventType {
		return nil
	}

	var payload KycPayload
	if err := envelope.UnmarshalPayload(&payload); err != nil {
		return err
	}

	alreadyProcessed, err := c.dedupeStore.MarkProcessed(ctx, envelope.EventID)
	if err != nil {
		return err
	}
	if alreadyProcessed {
		return nil
	}

	custID, err := uuid.Parse(payload.CustomerID)
	if err != nil {
		c.log.Error("invalid customer_id in kyc event", "err", err)
		return err
	}

	c.log.Info("KYC consumer получил событие", "type", envelope.EventType, "customer_id", custID)

	switch envelope.EventType {
	case KycSubmittedEventType:

		return c.applyKyc.MarkOnKYC(ctx, custID)
	case KycApprovedEventType:

		return c.applyKyc.Apply(ctx, app.ApplyKycDecisionInput{CustomerID: custID, Approved: true})
	case KycRejectedEventType:

		return c.applyKyc.Apply(ctx, app.ApplyKycDecisionInput{CustomerID: custID, Approved: false})
	}
	
	return nil
}