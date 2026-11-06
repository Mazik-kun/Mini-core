package app

import (
	"context"
	"github.com/google/uuid"
)

type MockAccountClient struct{}

func (m *MockAccountClient) GetAccount(ctx context.Context, accountID uuid.UUID) (string, string, error) {
	return "GEL", "ACTIVE", nil
}