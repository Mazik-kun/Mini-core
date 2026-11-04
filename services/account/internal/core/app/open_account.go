package app

import (
	"context"
	"fmt"
	"math/rand"
	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type OpenAccountUseCase struct {
	repo         domain.AccountRepository
	customerCli  domain.CustomerClient
}

func NewOpenAccountUseCase(repo domain.AccountRepository, cli domain.CustomerClient) *OpenAccountUseCase {
	return &OpenAccountUseCase{repo: repo, customerCli: cli}
}

func (uc *OpenAccountUseCase) OpenAccount(ctx context.Context, customerID uuid.UUID, currency string, accType domain.Type) (domain.Account, error) {
	status, err := uc.customerCli.GetCustomerStatus(ctx, customerID)
	if err != nil {
		return domain.Account{}, fmt.Errorf("failed to check customer status: %w", err)
	}
	if status != "CUSTOMER_STATUS_ACTIVE" {
		return domain.Account{}, domain.ErrCustomerNotActive
	}

	count, err := uc.repo.CountByCurrency(ctx, customerID, currency)
	if err != nil {
		return domain.Account{}, err
	}
	if count >= 3 {
		return domain.Account{}, domain.ErrLimitExceeded
	}

	accNum := fmt.Sprintf("GE%016d", rand.Int63n(9999999999999999))

	newAcc := domain.Account{
		ID:            uuid.New(),
		CustomerID:    customerID,
		AccountNumber: accNum,
		Currency:      currency,
		Type:          accType,
		Status:        domain.StatusActive,
	}
	return uc.repo.Create(ctx, newAcc)
}