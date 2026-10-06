package app

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

// fakeRepo — ручная реализация CustomerRepository для тестов.
// Хранит данные в map, не ходит в БД.
type fakeRepo struct {
	customers map[uuid.UUID]domain.Customer
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{customers: map[uuid.UUID]domain.Customer{}}
}

func (r *fakeRepo) Create(_ context.Context, c domain.Customer) (domain.Customer, error) {
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

func (r *fakeRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.Status) error {
	return nil
}

// --- Тесты ---

func TestGetCustomer_ClientRequestsSelf_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	userID := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, UserID: userID, FullName: "Ivan"}

	uc := NewGetCustomerUseCase(repo)
	got, err := uc.GetCustomer(context.Background(), GetCustomerInput{
		CustomerID:    id,
		RequesterID:   id,
		RequesterRole: domain.RoleClient,
	})

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
	repo.customers[otherID] = domain.Customer{ID: otherID}

	uc := NewGetCustomerUseCase(repo)
	_, err := uc.GetCustomer(context.Background(), GetCustomerInput{
		CustomerID:    otherID,
		RequesterID:   uuid.New(), // другой id
		RequesterRole: domain.RoleClient,
	})

	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestGetCustomer_OfficerRequestsAny_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, FullName: "Petrov"}

	uc := NewGetCustomerUseCase(repo)
	got, err := uc.GetCustomer(context.Background(), GetCustomerInput{
		CustomerID:    id,
		RequesterID:   uuid.New(), // офицер — другой id, но роль позволяет
		RequesterRole: domain.RoleOfficer,
	})

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
	_, err := uc.GetCustomer(context.Background(), GetCustomerInput{
		CustomerID:    id,
		RequesterID:   id,
		RequesterRole: domain.RoleClient,
	})

	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestGetCustomer_InvalidID(t *testing.T) {
	repo := newFakeRepo()
	uc := NewGetCustomerUseCase(repo)

	_, err := uc.GetCustomer(context.Background(), GetCustomerInput{
		CustomerID:    uuid.Nil(),
		RequesterID:   uuid.New(),
		RequesterRole: domain.RoleClient,
	})

	if !errors.Is(err, domain.ErrInvalidCustomerID) {
		t.Errorf("want ErrInvalidCustomerID, got %v", err)
	}
}