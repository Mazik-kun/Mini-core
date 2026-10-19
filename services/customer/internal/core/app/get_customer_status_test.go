package app

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

func TestGetCustomerStatus_ClientRequestsSelf_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	userID := uuid.New()
	wantStatus := domain.StatusActive
	repo.customers[id] = domain.Customer{ID: id, UserID: userID, Status: domain.Status(wantStatus)}

	uc := NewGetCustomerStatusUseCase(repo)
	ctx := contextWithClaims(userID, &id, roles.RoleClient)

	got, err := uc.GetCustomerStatus(ctx, GetCustomerStatusInput{CustomerID: id})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != domain.Status(wantStatus) {
		t.Errorf("got status %q, want %q", got, wantStatus)
	}
}

func TestGetCustomerStatus_ClientRequestsOther_AccessDenied(t *testing.T) {
	repo := newFakeRepo()
	otherID := uuid.New()
	ownCustomerID := uuid.New()
	repo.customers[otherID] = domain.Customer{ID: otherID, Status: domain.StatusActive}

	uc := NewGetCustomerStatusUseCase(repo)
	ctx := contextWithClaims(uuid.New(), &ownCustomerID, roles.RoleClient)

	_, err := uc.GetCustomerStatus(ctx, GetCustomerStatusInput{CustomerID: otherID})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestGetCustomerStatus_OfficerRequestsAny_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	wantStatus := domain.StatusBlocked
	repo.customers[id] = domain.Customer{ID: id, Status: domain.Status(wantStatus)}

	uc := NewGetCustomerStatusUseCase(repo)
	ctx := contextWithClaims(uuid.New(), nil, roles.RoleOfficer)

	got, err := uc.GetCustomerStatus(ctx, GetCustomerStatusInput{CustomerID: id})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != domain.Status(wantStatus) {
		t.Errorf("got status %q, want %q", got, wantStatus)
	}
}

func TestGetCustomerStatus_NotFound(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	uc := NewGetCustomerStatusUseCase(repo)
	ctx := contextWithClaims(id, &id, roles.RoleClient)

	_, err := uc.GetCustomerStatus(ctx, GetCustomerStatusInput{CustomerID: id})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestGetCustomerStatus_InvalidID(t *testing.T) {
	repo := newFakeRepo()
	uc := NewGetCustomerStatusUseCase(repo)
	ctx := contextWithClaims(uuid.New(), nil, roles.RoleClient)

	_, err := uc.GetCustomerStatus(ctx, GetCustomerStatusInput{CustomerID: uuid.Nil})
	if !errors.Is(err, domain.ErrInvalidCustomerID) {
		t.Errorf("want ErrInvalidCustomerID, got %v", err)
	}
}
