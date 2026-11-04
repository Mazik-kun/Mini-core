package domain

import (
	"context"
	"github.com/google/uuid"
)

type AccountRepository interface {
	Create(ctx context.Context, acc Account) (Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (Account, error)
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]Account, error)
	CountByCurrency(ctx context.Context, customerID uuid.UUID, currency string) (int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (Account, error)
}

type CustomerClient interface {
	GetCustomerStatus(ctx context.Context, customerID uuid.UUID) (string, error)
}