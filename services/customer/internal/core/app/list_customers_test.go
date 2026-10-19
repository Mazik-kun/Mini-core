package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

const (
	testDefaultPageSize = 50
	testMaxPageSize     = 100
)

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

func newListCustomersUC(repo domain.CustomerRepository) *ListCustomersUseCase {
	return NewListCustomersUseCase(repo, testDefaultPageSize, testMaxPageSize)
}

func officerContext() context.Context {
	return contextWithClaims(uuid.New(), nil, roles.RoleOfficer)
}

func adminContext() context.Context {
	return contextWithClaims(uuid.New(), nil, roles.RoleAdmin)
}

func clientContext() context.Context {
	return contextWithClaims(uuid.New(), nil, roles.RoleClient)
}

func TestListCustomers_ClientDenied(t *testing.T) {
	uc := newListCustomersUC(newListRepo(0))
	_, err := uc.ListCustomers(clientContext(), ListCustomersInput{})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestListCustomers_InvalidStatus(t *testing.T) {
	uc := newListCustomersUC(newListRepo(0))
	_, err := uc.ListCustomers(officerContext(), ListCustomersInput{
		Statuses: []domain.Status{"WHATEVER"},
	})
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("want ErrInvalidStatus, got %v", err)
	}
}

func TestListCustomers_InvalidToken(t *testing.T) {
	uc := newListCustomersUC(newListRepo(0))
	for _, token := range []string{"!!!not-base64!!!", "YWJj"} {
		_, err := uc.ListCustomers(officerContext(), ListCustomersInput{PageToken: token})
		if !errors.Is(err, domain.ErrInvalidPageToken) {
			t.Errorf("token %q: want ErrInvalidPageToken, got %v", token, err)
		}
	}
}

func TestListCustomers_HasNextPage(t *testing.T) {
	repo := newListRepo(5)
	uc := newListCustomersUC(repo)

	out, err := uc.ListCustomers(officerContext(), ListCustomersInput{PageSize: 3})
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
	uc := newListCustomersUC(newListRepo(3))
	out, err := uc.ListCustomers(adminContext(), ListCustomersInput{PageSize: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Customers) != 3 || out.NextPageToken != "" {
		t.Errorf("got %d customers, token %q; want 3 and empty token", len(out.Customers), out.NextPageToken)
	}
}

func TestListCustomers_DefaultAndMaxPageSize(t *testing.T) {
	repo := newListRepo(0)
	uc := newListCustomersUC(repo)

	_, _ = uc.ListCustomers(officerContext(), ListCustomersInput{})
	if repo.gotFilter.Limit != testDefaultPageSize+1 {
		t.Errorf("default: limit = %d, want %d", repo.gotFilter.Limit, testDefaultPageSize+1)
	}

	_, _ = uc.ListCustomers(officerContext(), ListCustomersInput{PageSize: 10_000})
	if repo.gotFilter.Limit != testMaxPageSize+1 {
		t.Errorf("max: limit = %d, want %d", repo.gotFilter.Limit, testMaxPageSize+1)
	}
}

func TestListCustomers_EmptyStatusesPassedAsNil(t *testing.T) {
	repo := newListRepo(0)
	uc := newListCustomersUC(repo)
	_, _ = uc.ListCustomers(officerContext(), ListCustomersInput{Statuses: []domain.Status{}})
	if repo.gotFilter.Statuses != nil {
		t.Errorf("want nil statuses, got %v", repo.gotFilter.Statuses)
	}
}
