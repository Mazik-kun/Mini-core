package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

// validUpdateProfileInput возвращает валидный набор данных для UpdateProfile.
// Если в вашем проекте другие требования к валидации — поправьте значения.
func validUpdateProfileInput(customerID uuid.UUID) UpdateProfileInput {
	return UpdateProfileInput{
		CustomerID:    customerID,
		RequesterID:   customerID,
		RequesterRole: domain.RoleClient,
		FullName:      "Ivan Ivanov",
		BirthDate:     pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		Address:       "Moscow, Tverskaya 1",
		PhoneNumber:   "+79001234567",
		Citizenship:   "RU",
	}
}

func TestUpdateProfile_ClientUpdatesSelf_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	repo.customers[id] = domain.Customer{
		ID:     id,
		UserID: id,
		Status: domain.StatusNew, // статус, который разрешает редактирование
	}

	uc := NewUpdateProfileUseCase(repo)
	in := validUpdateProfileInput(id)

	got, err := uc.UpdateProfile(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.FullName != in.FullName {
		t.Errorf("FullName = %q, want %q", got.FullName, in.FullName)
	}
	if got.Address != in.Address {
		t.Errorf("Address = %q, want %q", got.Address, in.Address)
	}
	if got.BirthDate != in.BirthDate {
		t.Errorf("BirthDate = %v, want %v", got.BirthDate, in.BirthDate)
	}
	if got.PhoneNumber != in.PhoneNumber {
		t.Errorf("PhoneNumber = %q, want %q", got.PhoneNumber, in.PhoneNumber)
	}
	if got.Citizenship != in.Citizenship {
		t.Errorf("Citizenship = %q, want %q", got.Citizenship, in.Citizenship)
	}

	// Если начальный статус позволяет переход в ProfileFilled,
	// после обновления он должен стать ProfileFilled.
	if got.Status != domain.StatusProfileFilled {
		t.Errorf("Status = %v, want %v", got.Status, domain.StatusProfileFilled)
	}
}

func TestUpdateProfile_ClientUpdatesOther_AccessDenied(t *testing.T) {
	repo := newFakeRepo()
	otherID := uuid.New()

	repo.customers[otherID] = domain.Customer{
		ID:     otherID,
		Status: domain.StatusNew,
	}

	uc := NewUpdateProfileUseCase(repo)
	in := validUpdateProfileInput(otherID)
	in.RequesterID = uuid.New() // другой id
	in.RequesterRole = domain.RoleClient

	_, err := uc.UpdateProfile(context.Background(), in)
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestUpdateProfile_OfficerUpdatesAny_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	repo.customers[id] = domain.Customer{
		ID:     id,
		Status: domain.StatusNew,
	}

	uc := NewUpdateProfileUseCase(repo)
	in := validUpdateProfileInput(id)
	in.RequesterID = uuid.New() // офицер — другой id, но роль позволяет
	in.RequesterRole = domain.RoleOfficer

	got, err := uc.UpdateProfile(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != id {
		t.Errorf("got id %v, want %v", got.ID, id)
	}
}

func TestUpdateProfile_NotFound(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	uc := NewUpdateProfileUseCase(repo)
	in := validUpdateProfileInput(id)

	_, err := uc.UpdateProfile(context.Background(), in)
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestUpdateProfile_ProfileLocked(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	// "blocked" — статус, который не разрешает редактирование профиля.
	// Если в domain есть константа (например, domain.StatusBlocked) — используйте её.
	repo.customers[id] = domain.Customer{
		ID:     id,
		Status: domain.Status("blocked"),
	}

	uc := NewUpdateProfileUseCase(repo)
	in := validUpdateProfileInput(id)

	_, err := uc.UpdateProfile(context.Background(), in)
	if !errors.Is(err, domain.ErrProfileLocked) {
		t.Errorf("want ErrProfileLocked, got %v", err)
	}
}

func TestUpdateProfile_ValidationErrors(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()

	repo.customers[id] = domain.Customer{
		ID:     id,
		Status: domain.StatusNew,
	}

	uc := NewUpdateProfileUseCase(repo)

	tests := []struct {
		name    string
		mutate  func(*UpdateProfileInput)
		wantErr error
	}{
		{
			name:    "invalid address",
			mutate:  func(in *UpdateProfileInput) { in.Address = "" },
			wantErr: domain.ErrInvalidAddress,
		},
		{
			name:    "invalid phone number",
			mutate:  func(in *UpdateProfileInput) { in.PhoneNumber = "123" },
			wantErr: domain.ErrInvalidPhoneNumber,
		},
		{
			name:    "invalid birth date",
			mutate:  func(in *UpdateProfileInput) { in.BirthDate = pgtype.Date{Time: time.Now().AddDate(1, 0, 0), Valid: true} },
			wantErr: domain.ErrInvalidBirthDate,
		},
		{
			name:    "invalid full name",
			mutate:  func(in *UpdateProfileInput) { in.FullName = "" },
			wantErr: domain.ErrInvalidFullName,
		},
		{
			name:    "invalid citizenship",
			mutate:  func(in *UpdateProfileInput) { in.Citizenship = "" },
			wantErr: domain.ErrInvalidCitizenship,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validUpdateProfileInput(id)
			tt.mutate(&in)

			_, err := uc.UpdateProfile(context.Background(), in)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}