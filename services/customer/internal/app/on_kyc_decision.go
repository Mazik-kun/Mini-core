package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

// ApplyKycDecisionUseCase применяет вердикт KYC к статусу клиента.
type ApplyKycDecisionUseCase struct {
	repo domain.CustomerRepository
}

func NewApplyKycDecisionUseCase(repo domain.CustomerRepository) *ApplyKycDecisionUseCase {
	return &ApplyKycDecisionUseCase{repo: repo}
}

type ApplyKycDecisionInput struct {
	CustomerID uuid.UUID
	Approved   bool
}

func (uc *ApplyKycDecisionUseCase) Apply(ctx context.Context, in ApplyKycDecisionInput) error {
	if in.CustomerID == uuid.Nil {
		return domain.ErrInvalidCustomerID
	}

	customer, err := uc.repo.GetByID(ctx, in.CustomerID)
	if err != nil {
		return err
	}

	if in.Approved {
		return uc.transition(ctx, customer, domain.StatusActive)
	}
	return uc.transition(ctx, customer, domain.StatusRejected)
}

// transition идемпотентно меняет статус. Повторный вызов с уже достигнутым
// статусом — не ошибка (Kafka доставляет at-least-once).
func (uc *ApplyKycDecisionUseCase) transition(ctx context.Context, customer domain.Customer, target domain.Status) error {
	if customer.Status == target {
		return nil
	}
	if !customer.Status.CanTransitionTo(target) {
		return domain.ErrInvalidStatusTransition
	}
	return uc.repo.UpdateStatus(ctx, customer.ID, target)
}