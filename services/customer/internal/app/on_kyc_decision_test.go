
package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

func TestApplyKycDecision_Approved(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusOnKYC}

	uc := NewApplyKycDecisionUseCase(repo)

	if err := uc.Apply(context.Background(), ApplyKycDecisionInput{CustomerID: id, Approved: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.customers[id].Status != domain.StatusActive {
		t.Errorf("status = %q, want %q", repo.customers[id].Status, domain.StatusActive)
	}
}

func TestApplyKycDecision_Rejected(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusOnKYC}

	uc := NewApplyKycDecisionUseCase(repo)

	if err := uc.Apply(context.Background(), ApplyKycDecisionInput{CustomerID: id, Approved: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.customers[id].Status != domain.StatusRejected {
		t.Errorf("status = %q, want %q", repo.customers[id].Status, domain.StatusRejected)
	}
}

// Повторная доставка kyc.application_approved — не ошибка.
func TestApplyKycDecision_Approved_Idempotent(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusActive}

	uc := NewApplyKycDecisionUseCase(repo)

	if err := uc.Apply(context.Background(), ApplyKycDecisionInput{CustomerID: id, Approved: true}); err != nil {
		t.Errorf("want nil, got %v", err)
	}
}

func TestApplyKycDecision_Rejected_Idempotent(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusRejected}

	uc := NewApplyKycDecisionUseCase(repo)

	if err := uc.Apply(context.Background(), ApplyKycDecisionInput{CustomerID: id, Approved: false}); err != nil {
		t.Errorf("want nil, got %v", err)
	}
}

func TestApplyKycDecision_InvalidTransition(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusNew}

	uc := NewApplyKycDecisionUseCase(repo)

	err := uc.Apply(context.Background(), ApplyKycDecisionInput{CustomerID: id, Approved: true})
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Errorf("want ErrInvalidStatusTransition, got %v", err)
	}
}

func TestApplyKycDecision_NotFound(t *testing.T) {
	repo := newFakeRepo()
	uc := NewApplyKycDecisionUseCase(repo)

	err := uc.Apply(context.Background(), ApplyKycDecisionInput{CustomerID: uuid.New(), Approved: true})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestApplyKycDecision_InvalidID(t *testing.T) {
	repo := newFakeRepo()
	uc := NewApplyKycDecisionUseCase(repo)

	err := uc.Apply(context.Background(), ApplyKycDecisionInput{CustomerID: uuid.Nil, Approved: true})
	if !errors.Is(err, domain.ErrInvalidCustomerID) {
		t.Errorf("want ErrInvalidCustomerID, got %v", err)
	}
}