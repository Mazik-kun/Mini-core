package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CustomerLinker — минимальный набор методов, нужный этому use case.
// Реальный *postgres.UserRepository ему удовлетворяет, но интерфейс
// UserRepository в domain мы не расширяем, чтобы не ломать существующие тесты.
type CustomerLinker interface {
	UpdateCustomerID(ctx context.Context, userID, customerID uuid.UUID) error
}

type ApplyCustomerCreatedUseCase struct {
	users CustomerLinker
}

func NewApplyCustomerCreatedUseCase(users CustomerLinker) *ApplyCustomerCreatedUseCase {
	return &ApplyCustomerCreatedUseCase{users: users}
}

type ApplyCustomerCreatedInput struct {
	UserID     uuid.UUID
	CustomerID uuid.UUID
}

func (uc *ApplyCustomerCreatedUseCase) ApplyCustomerCreated(ctx context.Context, in ApplyCustomerCreatedInput) error {
	if in.UserID == uuid.Nil {
		return fmt.Errorf("apply customer.created: пустой user_id")
	}
	if in.CustomerID == uuid.Nil {
		return fmt.Errorf("apply customer.created: пустой customer_id")
	}
	return uc.users.UpdateCustomerID(ctx, in.UserID, in.CustomerID)
}