package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

// listRepo — fakeRepo с рабочим List: отдаёт первые Limit записей и запоминает фильтр.
type listRepo struct {
	*fakeRepo
	items     []domain.Customer
	gotFilter domain.ListFilter
}

func (r *listRepo) List(_ context.Context, f domain.ListFilter) ([]domain.Customer, error) {
	r.gotFilter = f
	return r.items[:min(f.Limit, len(r.items))], nil
}

func newListRepo(n int) *listRepo {
	items := make([]domain.Customer, n)
	for i := range items {
		items[i] = domain.Customer{ID: uuid.New()}
	}
	return &listRepo{fakeRepo: newFakeRepo(), items: items}
}

func TestListCustomers_ClientDenied(t *testing.T) {
	uc := NewListCustomersUseCase(newListRepo(0))
	_, err := uc.ListCustomers(context.Background(), ListCustomersInput{RequesterRole: domain.RoleClient})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestListCustomers_InvalidStatus(t *testing.T) {
	uc := NewListCustomersUseCase(newListRepo(0))
	_, err := uc.ListCustomers(context.Background(), ListCustomersInput{
		RequesterRole: domain.RoleOfficer,
		Statuses:      []domain.Status{"WHATEVER"},
	})
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("want ErrInvalidStatus, got %v", err)
	}
}

func TestListCustomers_InvalidToken(t *testing.T) {
	uc := NewListCustomersUseCase(newListRepo(0))
	for _, token := range []string{"!!!not-base64!!!", "YWJj"} { // мусор и неверная длина
		_, err := uc.ListCustomers(context.Background(), ListCustomersInput{
			RequesterRole: domain.RoleOfficer,
			PageToken:     token,
		})
		if !errors.Is(err, domain.ErrInvalidPageToken) {
			t.Errorf("token %q: want ErrInvalidPageToken, got %v", token, err)
		}
	}
}

func TestListCustomers_HasNextPage(t *testing.T) {
	repo := newListRepo(5)
	uc := NewListCustomersUseCase(repo)

	out, err := uc.ListCustomers(context.Background(), ListCustomersInput{
		RequesterRole: domain.RoleOfficer,
		PageSize:      3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotFilter.Limit != 4 {
		t.Errorf("repo limit = %d, want 4 (size+1)", repo.gotFilter.Limit)
	}
	if len(out.Customers) != 3 {
		t.Fatalf("got %d customers, want 3", len(out.Customers))
	}
	cursor, err := decodePageToken(out.NextPageToken)
	if err != nil || cursor != out.Customers[2].ID {
		t.Errorf("token must point to last returned customer, got %v (err %v)", cursor, err)
	}
}

func TestListCustomers_LastPage_NoToken(t *testing.T) {
	uc := NewListCustomersUseCase(newListRepo(3))
	out, err := uc.ListCustomers(context.Background(), ListCustomersInput{
		RequesterRole: domain.RoleAdmin,
		PageSize:      3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Customers) != 3 || out.NextPageToken != "" {
		t.Errorf("got %d customers, token %q; want 3 and empty token", len(out.Customers), out.NextPageToken)
	}
}

func TestListCustomers_DefaultAndMaxPageSize(t *testing.T) {
	repo := newListRepo(0)
	uc := NewListCustomersUseCase(repo)

	_, _ = uc.ListCustomers(context.Background(), ListCustomersInput{RequesterRole: domain.RoleOfficer})
	if repo.gotFilter.Limit != defaultPageSize+1 {
		t.Errorf("default: limit = %d, want %d", repo.gotFilter.Limit, defaultPageSize+1)
	}

	_, _ = uc.ListCustomers(context.Background(), ListCustomersInput{RequesterRole: domain.RoleOfficer, PageSize: 10_000})
	if repo.gotFilter.Limit != maxPageSize+1 {
		t.Errorf("max: limit = %d, want %d", repo.gotFilter.Limit, maxPageSize+1)
	}
}

func TestListCustomers_EmptyStatusesPassedAsNil(t *testing.T) {
	repo := newListRepo(0)
	uc := NewListCustomersUseCase(repo)
	_, _ = uc.ListCustomers(context.Background(), ListCustomersInput{
		RequesterRole: domain.RoleOfficer,
		Statuses:      []domain.Status{},
	})
	if repo.gotFilter.Statuses != nil {
		t.Errorf("want nil statuses, got %v", repo.gotFilter.Statuses)
	}
}