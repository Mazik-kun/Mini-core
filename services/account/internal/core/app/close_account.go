package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type CloseAccountUseCase struct {
	repo domain.AccountRepository
}

func NewCloseAccountUseCase(repo domain.AccountRepository) *CloseAccountUseCase {
	return &CloseAccountUseCase{repo: repo}
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

	// TODO: 3. Проверить баланс в Ledger Service. Если balance > 0, вернуть domain.ErrBalanceNotZero.
	// Пока (до реализации Ledger) мы пропускаем эту проверку.

	// 4. Меняем статус
	return uc.repo.UpdateStatus(ctx, acc.ID, domain.StatusClosed)
}