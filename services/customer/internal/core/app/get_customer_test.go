package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

// fakeRepo — ручная реализация CustomerRepository для тестов.
type fakeRepo struct {
	customers map[uuid.UUID]domain.Customer
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{customers: map[uuid.UUID]domain.Customer{}}
}

func (r *fakeRepo) Create(_ context.Context, c domain.Customer) (domain.Customer, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if _, ok := r.customers[c.ID]; ok {
		return domain.Customer{}, domain.ErrAlreadyExists
	}
	r.customers[c.ID] = c
	return c, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Customer, error) {
	c, ok := r.customers[id]
	if !ok {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	return c, nil
}

func (r *fakeRepo) GetByUserID(_ context.Context, userID uuid.UUID) (domain.Customer, error) {
	for _, c := range r.customers {
		if c.UserID == userID {
			return c, nil
		}
	}
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *fakeRepo) Update(_ context.Context, c domain.Customer) (domain.Customer, error) {
	if _, ok := r.customers[c.ID]; !ok {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	r.customers[c.ID] = c
	return c, nil
}

func (r *fakeRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.Customer, error) {
	return nil, nil
}

func (r *fakeRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.Status) error {
	c, ok := r.customers[id]
	if !ok {
		return domain.ErrCustomerNotFound
	}
	c.Status = status
	r.customers[id] = c
	return nil
}

// contextWithClaims собирает context так же, как это делает UnaryAuthInterceptor.
func contextWithClaims(userID uuid.UUID, customerID *uuid.UUID, role string) context.Context {
	return authjwt.WithClaims(context.Background(), authjwt.Claims{
		Subject:    userID,
		Role:       role,
		CustomerID: customerID,
	})
}

// --- Тесты ---

func TestGetCustomer_ClientRequestsSelf_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	userID := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, UserID: userID, FullName: "Ivan"}

	uc := NewGetCustomerUseCase(repo)
	ctx := contextWithClaims(userID, &id, roles.RoleClient)

	got, err := uc.GetCustomer(ctx, GetCustomerInput{CustomerID: id})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != id {
		t.Errorf("got id %v, want %v", got.ID, id)
	}
}

func TestGetCustomer_ClientRequestsOther_AccessDenied(t *testing.T) {
	repo := newFakeRepo()
	otherID := uuid.New()
	ownCustomerID := uuid.New()
	repo.customers[otherID] = domain.Customer{ID: otherID}

	uc := NewGetCustomerUseCase(repo)
	ctx := contextWithClaims(uuid.New(), &ownCustomerID, roles.RoleClient)

	_, err := uc.GetCustomer(ctx, GetCustomerInput{CustomerID: otherID})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestGetCustomer_OfficerRequestsAny_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, FullName: "Petrov"}

	uc := NewGetCustomerUseCase(repo)
	ctx := contextWithClaims(uuid.New(), nil, roles.RoleOfficer)

	got, err := uc.GetCustomer(ctx, GetCustomerInput{CustomerID: id})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != id {
		t.Errorf("got id %v, want %v", got.ID, id)
	}
}

func TestGetCustomer_NotFound(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	uc := NewGetCustomerUseCase(repo)
	ctx := contextWithClaims(id, &id, roles.RoleClient)

	_, err := uc.GetCustomer(ctx, GetCustomerInput{CustomerID: id})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestGetCustomer_InvalidID(t *testing.T) {
	repo := newFakeRepo()
	uc := NewGetCustomerUseCase(repo)
	ctx := contextWithClaims(uuid.New(), nil, roles.RoleClient)

	_, err := uc.GetCustomer(ctx, GetCustomerInput{CustomerID: uuid.Nil})
	if !errors.Is(err, domain.ErrInvalidCustomerID) {
		t.Errorf("want ErrInvalidCustomerID, got %v", err)
	}
}
