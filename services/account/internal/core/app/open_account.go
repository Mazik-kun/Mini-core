package app

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type OpenAccountUseCase struct {
	repo        domain.AccountRepository
	customerCli domain.CustomerClient
}

func NewOpenAccountUseCase(repo domain.AccountRepository, cli domain.CustomerClient) *OpenAccountUseCase {
	return &OpenAccountUseCase{repo: repo, customerCli: cli}
}

// OpenAccount - открытие счёта по запросу (gRPC): сначала проверяем, что клиент ACTIVE.
// Для вызова customer-сервиса нужен JWT пользователя в контексте (см. ForwardAuthUnary).
func (uc *OpenAccountUseCase) OpenAccount(ctx context.Context, customerID uuid.UUID, currency string, accType domain.Type) (domain.Account, error) {
	status, err := uc.customerCli.GetCustomerStatus(ctx, customerID)
	if err != nil {
		return domain.Account{}, fmt.Errorf("failed to check customer status: %w", err)
	}
	if status != "CUSTOMER_STATUS_ACTIVE" {
		return domain.Account{}, domain.ErrCustomerNotActive
	}
	return uc.create(ctx, customerID, currency, accType)
}

// OpenAccountForApprovedCustomer - открытие счёта по событию kyc.application_approved.
// Событие уже означает, что клиент одобрен, а JWT в обработчике Kafka нет, поэтому
// customer-сервис не вызываем (это же убирает гонку со сменой статуса на ACTIVE).
func (uc *OpenAccountUseCase) OpenAccountForApprovedCustomer(ctx context.Context, customerID uuid.UUID, currency string, accType domain.Type) (domain.Account, error) {
	return uc.create(ctx, customerID, currency, accType)
}

func (uc *OpenAccountUseCase) create(ctx context.Context, customerID uuid.UUID, currency string, accType domain.Type) (domain.Account, error) {
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