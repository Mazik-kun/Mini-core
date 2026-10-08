package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

// CreateCustomerUseCase создаёт пустой профиль клиента по событию auth.user_registered.
// Поля профиля заполняются позже через UpdateProfile.
type CreateCustomerUseCase struct {
	repo domain.CustomerRepository
}

func NewCreateCustomerUseCase(repo domain.CustomerRepository) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{repo: repo}
}

type CreateCustomerInput struct {
	UserID uuid.UUID
}

func (uc *CreateCustomerUseCase) CreateCustomer(ctx context.Context, in CreateCustomerInput) (domain.Customer, error) {
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
	return created, nil
}