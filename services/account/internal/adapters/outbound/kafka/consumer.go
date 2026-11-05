package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Mazik-kun/mini-core/services/account/internal/adapters/outbound/postgres"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type EventEnvelope struct {
	EventType string `json:"event_type"`
	Payload json.RawMessage `json:"payload"`
}

type KycApprovedPayload struct{
	ApplicationID string `json:"application_id"`
	CustomerID string `json:"customer_id"`
}

type Consumer struct {
	reader *kafka.Reader
	openUC *app.OpenAccountUseCase
	listUC *app.ListAccountsUseCase
	log *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string,repo *postgres.AccountRepo, openUC *app.OpenAccountUseCase,listUC *app.ListAccountsUseCase, log *slog.Logger) *Consumer{
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic: topic,
		GroupID: groupID,
		MinBytes: 10,
		MaxBytes: 10*1024*1024, // 10MB
	})
	return &Consumer{
		reader: r,
		openUC: openUC,
		listUC: listUC,
		log: log,
	}
}

func (c *Consumer) Run(ctx context.Context) error{
	c.log.Info("kafka consumer started", "topic", c.reader.Config().Topic)

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil{
			if ctx.Err() != nil{
				return nil
			}
			c.log.Error("failed to read kafka message", "error", err)
			continue
		}
		c.handleMessage(ctx, msg.Value)
	}
}
func (c *Consumer) Close()error{
	return c.reader.Close()
}

func (c *Consumer)handleMessage(ctx context.Context, payload []byte){
	var env EventEnvelope
	if err := json.Unmarshal(payload, &env); err != nil{
		c.log.Error("failed to unmarshal event envelope", "error", err)
		return
	}

	if env.EventType != "kyc.application_approved"{
		return
	}
	var kycPayload KycApprovedPayload
	if err := json.Unmarshal(env.Payload, &kycPayload); err != nil{
		c.log.Error("failed to unmarshal kyc payload", "error", err)
		return
	}
	cid, err := uuid.Parse(kycPayload.CustomerID)
	if err != nil{
		c.log.Error("invalid customer_id in kafka event", "error", err)
		return
	}

	existingAccounts, err := c.listUC.ListByCustomer(ctx, cid)
	if err == nil && len(existingAccounts) > 0{
		c.log.Info("account already exists, skipping auto-open", "customer_id", cid)
		return
	}

	acc, err := c.openUC.OpenAccount(ctx, cid, "GEL", domain.TypeChecking)
	if err != nil{
		c.log.Error("failed to auto-open account", "customer_id", cid, "error", err)
		return
	}
	c.log.Info("SUCCESS: auto-opened account via kafka", "customer_id", cid, "account_id", acc.ID)
}