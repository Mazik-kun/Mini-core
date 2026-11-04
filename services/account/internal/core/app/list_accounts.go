package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type ListAccountsUseCase struct {
	repo domain.AccountRepository
}

func NewListAccountsUseCase(repo domain.AccountRepository) *ListAccountsUseCase {
	return &ListAccountsUseCase{repo: repo}
}

func (uc *ListAccountsUseCase) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.Account, error) {
	return uc.repo.ListByCustomer(ctx, customerID)
}