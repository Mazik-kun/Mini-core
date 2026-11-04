package app

import (
	"context"
	"github.com/google/uuid"
)

type MockCustomerClient struct{}

func (m *MockCustomerClient) GetCustomerStatus(ctx context.Context, customerID uuid.UUID) (string, error) {
	return "CUSTOMER_STATUS_ACTIVE", nil
}