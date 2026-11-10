package app

import (
	"context"

	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ListTransactionsUseCase struct {
	repo domain.LedgerRepository
}

func NewListTransactionsUseCase(repo domain.LedgerRepository) * ListTransactionsUseCase{
	return &ListTransactionsUseCase{repo: repo}
}

func (uc *ListTransactionsUseCase) ListTransactions(
	ctx context.Context,
	accountID uuid.UUID,
	fromTime, toTime pgtype.Timestamptz,
	limit, offset int32,
)([]domain.Transaction, error){
	return uc.repo.ListTransactions(ctx, accountID, fromTime, toTime, limit, offset)
}