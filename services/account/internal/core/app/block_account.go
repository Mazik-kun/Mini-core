package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type BlockAccountUseCase struct {
	repo domain.AccountRepository
}

func NewBlockAccountUseCase(repo domain.AccountRepository) *BlockAccountUseCase {
	return &BlockAccountUseCase{repo: repo}
}

func (uc *BlockAccountUseCase) BlockAccount(ctx context.Context, accountID uuid.UUID) (domain.Account, error) {
	// 1. Получаем счет
	acc, err := uc.repo.GetByID(ctx, accountID)
	if err != nil {
		return domain.Account{}, err
	}

	// 2. Проверяем статус (нельзя заблокировать закрытый или уже заблокированный)
	if acc.Status == domain.StatusClosed {
		return domain.Account{}, fmt.Errorf("%w: account is closed", domain.ErrInvalidStatus)
	}
	if acc.Status == domain.StatusBlocked {
		return domain.Account{}, fmt.Errorf("%w: account is already blocked", domain.ErrInvalidStatus)
	}

	// 3. Меняем статус
	return uc.repo.UpdateStatus(ctx, acc.ID, domain.StatusBlocked)
}