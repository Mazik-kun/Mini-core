package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
)

type GetTransactionUseCase struct {
	repo domain.LedgerRepository
}

func NewGetTransactionUseCase(repo domain.LedgerRepository) *GetTransactionUseCase {
	return &GetTransactionUseCase{repo: repo}
}

func (uc *GetTransactionUseCase) GetTransaction(ctx context.Context, id uuid.UUID) (domain.Transaction, error) {
	return uc.repo.GetTransactionByID(ctx, id)
}