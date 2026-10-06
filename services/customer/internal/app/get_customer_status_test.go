package app

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

func TestGetCustomerStatus_ClientRequestsSelf_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	userID := uuid.New()
	wantStatus := domain.Status("active")

	repo.customers[id] = domain.Customer{
		ID:     id,
		UserID: userID,
		Status: wantStatus,
	}

	uc := NewGetCustomerStatusUseCase(repo)
	got, err := uc.GetCustomerStatus(context.Background(), GetCustomerStatusInput{
		CustomerID:    id,
		RequesterID:   id,
		RequesterRole: domain.RoleClient,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wantStatus {
		t.Errorf("got status %q, want %q", got, wantStatus)
	}
}

func TestGetCustomerStatus_ClientRequestsOther_AccessDenied(t *testing.T) {
	repo := newFakeRepo()
	otherID := uuid.New()

	repo.customers[otherID] = domain.Customer{
		ID:     otherID,
		Status: domain.Status("active"),
	}

	uc := NewGetCustomerStatusUseCase(repo)
	_, err := uc.GetCustomerStatus(context.Background(), GetCustomerStatusInput{
		CustomerID:    otherID,
		RequesterID:   uuid.New(), // другой id
		RequesterRole: domain.RoleClient,
	})

	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestGetCustomerStatus_OfficerRequestsAny_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	wantStatus := domain.Status("blocked")

	repo.customers[id] = domain.Customer{
		ID:     id,
		Status: wantStatus,
	}

	uc := NewGetCustomerStatusUseCase(repo)
	got, err := uc.GetCustomerStatus(context.Background(), GetCustomerStatusInput{
		CustomerID:    id,
		RequesterID:   uuid.New(), // офицер — другой id, но роль позволяет
		RequesterRole: domain.RoleOfficer,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wantStatus {
		t.Errorf("got status %q, want %q", got, wantStatus)
	}
}

func TestGetCustomerStatus_NotFound(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	uc := NewGetCustomerStatusUseCase(repo)
	_, err := uc.GetCustomerStatus(context.Background(), GetCustomerStatusInput{
		CustomerID:    id,
		RequesterID:   id,
		RequesterRole: domain.RoleClient,
	})

	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestGetCustomerStatus_InvalidID(t *testing.T) {
	repo := newFakeRepo()
	uc := NewGetCustomerStatusUseCase(repo)

	_, err := uc.GetCustomerStatus(context.Background(), GetCustomerStatusInput{
		CustomerID:    uuid.Nil(),
		RequesterID:   uuid.New(),
		RequesterRole: domain.RoleClient,
	})

	if !errors.Is(err, domain.ErrInvalidCustomerID) {
		t.Errorf("want ErrInvalidCustomerID, got %v", err)
	}
}