package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type GetBalanceUseCase struct {
	repo domain.AccountRepository
}

func NewGetBalanceUseCase(repo domain.AccountRepository) *GetBalanceUseCase {
	return &GetBalanceUseCase{repo: repo}
}

// BalanceDTO — структура для возврата баланса (пока Ledger не готов)
type BalanceDTO struct {
	Amount   int64
	Currency string
}

func (uc *GetBalanceUseCase) GetBalance(ctx context.Context, accountID uuid.UUID) (BalanceDTO, error) {
	// 1. Проверяем, что счет вообще существует
	acc, err := uc.repo.GetByID(ctx, accountID)
	if err != nil {
		return BalanceDTO{}, err
	}

	// TODO: Сходить по gRPC в Ledger Service за реальным балансом.
	// Пока Ledger не написан, отдаем 0.
	return BalanceDTO{
		Amount:   0,
		Currency: acc.Currency,
	}, nil
}