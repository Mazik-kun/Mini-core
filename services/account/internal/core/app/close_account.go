package app

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type CloseAccountUseCase struct {
	repo      domain.AccountRepository
	ledgerCli domain.LedgerClient
}

// Обновляем конструктор
func NewCloseAccountUseCase(repo domain.AccountRepository, ledgerCli domain.LedgerClient) *CloseAccountUseCase {
	return &CloseAccountUseCase{
		repo:      repo,
		ledgerCli: ledgerCli,
	}
}

func (uc *CloseAccountUseCase) CloseAccount(ctx context.Context, accountID uuid.UUID) (domain.Account, error) {
	// 1. Получаем счет
	acc, err := uc.repo.GetByID(ctx, accountID)
	if err != nil {
		return domain.Account{}, err
	}

	// 2. Проверяем статус
	if acc.Status == domain.StatusClosed {
		return domain.Account{}, fmt.Errorf("%w: account is already closed", domain.ErrInvalidStatus)
	}

	// 3. Проверяем баланс в Ledger
	balance, err := uc.ledgerCli.GetBalance(ctx, accountID)
	if err != nil {
		return domain.Account{}, fmt.Errorf("failed to check balance: %w", err)
	}
	log.Printf("DEBUG close account: id=%s balance=%d", accountID, balance) // TODO: убрать после отладки
	// != 0, а не > 0: счёт с отрицательным балансом закрывать тоже нельзя
	if balance != 0 {
		return domain.Account{}, domain.ErrBalanceNotZero
	}

	// 4. Меняем статус
	return uc.repo.UpdateStatus(ctx, acc.ID, domain.StatusClosed)
}