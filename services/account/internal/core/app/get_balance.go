package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type GetBalanceUseCase struct {
	repo      domain.AccountRepository
	ledgerCli domain.LedgerClient
}

func NewGetBalanceUseCase(repo domain.AccountRepository, ledgerCli domain.LedgerClient) *GetBalanceUseCase {
	return &GetBalanceUseCase{repo: repo, ledgerCli: ledgerCli}
}

type BalanceDTO struct {
	Amount   int64
	Currency string
}

func (uc *GetBalanceUseCase) GetBalance(ctx context.Context, accountID uuid.UUID) (BalanceDTO, error) {
	// 1. Проверяем, что счёт существует (иначе 404), и берём currency
	acc, err := uc.repo.GetByID(ctx, accountID)
	if err != nil {
		return BalanceDTO{}, err
	}

	// 2. Идём за балансом в Ledger
	amount, err := uc.ledgerCli.GetBalance(ctx, accountID)
	if err != nil {
		return BalanceDTO{}, fmt.Errorf("get balance from ledger: %w", err)
	}

	return BalanceDTO{
		Amount:   amount,
		Currency: acc.Currency, // Источник истины по валюте — Account
	}, nil
}