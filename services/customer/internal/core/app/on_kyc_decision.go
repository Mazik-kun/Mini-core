package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

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

func (uc *ApplyKycDecisionUseCase) transition(ctx context.Context, customer domain.Customer, target domain.Status) error {
	if customer.Status == target {
		return nil
	}
	if !customer.Status.CanTransitionTo(target) {
		return domain.ErrInvalidStatusTransition
	}
	return uc.repo.UpdateStatus(ctx, customer.ID, target)
}


func (uc *ApplyKycDecisionUseCase) MarkOnKYC(ctx context.Context, customerID uuid.UUID) error {
	if customerID == uuid.Nil {
		return domain.ErrInvalidCustomerID
	}
	customer, err := uc.repo.GetByID(ctx, customerID)
	if err != nil {
		return err
	}
	
	return uc.transition(ctx, customer, domain.StatusOnKYC)
}

