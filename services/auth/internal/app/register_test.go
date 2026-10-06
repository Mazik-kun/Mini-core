package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/auth/internal/app"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type fakeRepo struct {
	users map[string]domain.User
}

func (r *fakeRepo) Create(_ context.Context, u domain.User) (domain.User, error) {
	if _, ok := r.users[u.Email]; ok {
		return domain.User{}, domain.ErrEmailAlreadyExists
	}
	u.ID = uuid.New()
	r.users[u.Email] = u
	return u, nil
}

func (r *fakeRepo) GetByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := r.users[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error)               { return "hashed:" + plain, nil }
func (fakeHasher) Verify(plain, hash string) (bool, error)         { return hash == "hashed:"+plain, nil }

func newUseCase() (*app.RegisterUseCase, *fakeRepo) {
	repo := &fakeRepo{users: map[string]domain.User{}}
	return app.NewRegisterUseCase(repo, fakeHasher{}), repo
}

func TestRegister_Success(t *testing.T) {
	uc, _ := newUseCase()
	out, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "Test@Example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Email != "test@example.com" {
		t.Errorf("email should be lowercased, got %s", out.Email)
	}
	if out.UserID == uuid.Nil {
		t.Error("user id must not be nil")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	uc, _ := newUseCase()
	in := app.RegisterInput{Email: "a@b.com", Password: "password123"}
	if _, err := uc.Register(context.Background(), in); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := uc.Register(context.Background(), in)
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("want ErrEmailAlreadyExists, got %v", err)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	uc, _ := newUseCase()
	_, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "not-an-email",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Errorf("want ErrInvalidEmail, got %v", err)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	uc, _ := newUseCase()
	_, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "a@b.com",
		Password: "short",
	})
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Errorf("want ErrWeakPassword, got %v", err)
	}
}