package app

import (
	"context"
	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
)

type GetBalanceUseCase struct {
	repo domain.LedgerRepository
}

func NewGetBalanceUseCase(repo domain.LedgerRepository) *GetBalanceUseCase {
	return &GetBalanceUseCase{repo: repo}
}

func (uc *GetBalanceUseCase) GetBalance(ctx context.Context, accountID uuid.UUID) (domain.Balance, error) {
	return uc.repo.GetBalance(ctx, accountID)
}