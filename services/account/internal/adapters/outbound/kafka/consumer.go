package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type KYCApprovedEvent struct {
	CustomerID string `json:"customer_id"`
}

type Consumer struct {
	// тут поля для sarama/kafka-go, топик, группа
	openUC *app.OpenAccountUseCase
	log    *slog.Logger
}

func (c *Consumer) handleMessage(ctx context.Context, payload []byte) error {
	var event KYCApprovedEvent
	if err := json.Unmarshal(payload, &event); err != nil { return err }
	
	cid, _ := uuid.Parse(event.CustomerID)
	
	// Автоматически открываем счет в GEL
	acc, err := c.openUC.OpenAccount(ctx, cid, "GEL", domain.TypeChecking)
	if err != nil {
		c.log.Error("failed to auto open account", "customer_id", cid, "error", err)
		return err
	}
	c.log.Info("auto opened account", "customer_id", cid, "account_id", acc.ID)
	return nil
}