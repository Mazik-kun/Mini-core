package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

// validUpdateProfileInput возвращает валидный набор данных для UpdateProfile.
func validUpdateProfileInput(customerID uuid.UUID) UpdateProfileInput {
	return UpdateProfileInput{
		CustomerID:  customerID,
		FullName:    "Ivan Ivanov",
		BirthDate:   pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		Address:     "Moscow, Tverskaya 1",
		PhoneNumber: "+79001234567",
		Citizenship: "RU",
	}
}

func TestUpdateProfile_ClientUpdatesSelf_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	userID := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, UserID: userID, Status: domain.StatusNew}

	uc := NewUpdateProfileUseCase(repo)
	ctx := contextWithClaims(userID, &id, roles.RoleClient)
	in := validUpdateProfileInput(id)

	got, err := uc.UpdateProfile(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FullName != in.FullName {
		t.Errorf("FullName = %q, want %q", got.FullName, in.FullName)
	}
	if got.Status != domain.StatusProfileFilled {
		t.Errorf("Status = %v, want %v", got.Status, domain.StatusProfileFilled)
	}
}

func TestUpdateProfile_ClientUpdatesOther_AccessDenied(t *testing.T) {
	repo := newFakeRepo()
	otherID := uuid.New()
	ownCustomerID := uuid.New()
	repo.customers[otherID] = domain.Customer{ID: otherID, Status: domain.StatusNew}

	uc := NewUpdateProfileUseCase(repo)
	ctx := contextWithClaims(uuid.New(), &ownCustomerID, roles.RoleClient)

	_, err := uc.UpdateProfile(ctx, validUpdateProfileInput(otherID))
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("want ErrAccessDenied, got %v", err)
	}
}

func TestUpdateProfile_OfficerUpdatesAny_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusNew}

	uc := NewUpdateProfileUseCase(repo)
	ctx := contextWithClaims(uuid.New(), nil, roles.RoleOfficer)

	got, err := uc.UpdateProfile(ctx, validUpdateProfileInput(id))
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
	ctx := contextWithClaims(id, &id, roles.RoleClient)

	_, err := uc.UpdateProfile(ctx, validUpdateProfileInput(id))
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestUpdateProfile_ProfileLocked(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusBlocked}

	uc := NewUpdateProfileUseCase(repo)
	ctx := contextWithClaims(id, &id, roles.RoleClient)

	_, err := uc.UpdateProfile(ctx, validUpdateProfileInput(id))
	if !errors.Is(err, domain.ErrProfileLocked) {
		t.Errorf("want ErrProfileLocked, got %v", err)
	}
}

func TestUpdateProfile_ValidationErrors(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.customers[id] = domain.Customer{ID: id, Status: domain.StatusNew}

	uc := NewUpdateProfileUseCase(repo)
	ctx := contextWithClaims(id, &id, roles.RoleClient)

	tests := []struct {
		name    string
		mutate  func(*UpdateProfileInput)
		wantErr error
	}{
		{"invalid address", func(in *UpdateProfileInput) { in.Address = "" }, domain.ErrInvalidAddress},
		{"invalid phone number", func(in *UpdateProfileInput) { in.PhoneNumber = "123" }, domain.ErrInvalidPhoneNumber},
		{"invalid birth date", func(in *UpdateProfileInput) { in.BirthDate = pgtype.Date{Time: time.Now().AddDate(1, 0, 0), Valid: true} }, domain.ErrInvalidBirthDate},
		{"invalid full name", func(in *UpdateProfileInput) { in.FullName = "" }, domain.ErrInvalidFullName},
		{"invalid citizenship", func(in *UpdateProfileInput) { in.Citizenship = "" }, domain.ErrInvalidCitizenship},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validUpdateProfileInput(id)
			tt.mutate(&in)
			_, err := uc.UpdateProfile(ctx, in)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

var _ = context.Background // на всякий случай; можно удалить, если не используется