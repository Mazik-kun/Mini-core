package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/auth/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/auth/internal/core/domain"
)

func newCreateOfficerUseCase() (*app.CreateOfficerUseCase, *fakeRepo) {
	repo := &fakeRepo{users: map[string]domain.User{}}
	return app.NewCreateOfficerUseCase(repo, fakeHasher{}), repo
}

func TestCreateOfficer_ClientRoleForbidden(t *testing.T) {
	uc, _ := newCreateOfficerUseCase()
	ctx := authjwt.WithClaims(context.Background(), authjwt.Claims{
		Role: roles.RoleClient,
	})

	_, err := uc.CreateOfficer(ctx, app.CreateOfficerInput{
		Email:    "officer@bank.com",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("want ErrPermissionDenied, got %v", err)
	}
}

func TestCreateOfficer_AdminRoleSuccess(t *testing.T) {
	uc, repo := newCreateOfficerUseCase()
	ctx := authjwt.WithClaims(context.Background(), authjwt.Claims{
		Role: roles.RoleAdmin,
	})

	out, err := uc.CreateOfficer(ctx, app.CreateOfficerInput{
		Email:    "Officer@Bank.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Email != "officer@bank.com" {
		t.Errorf("email should be lowercased, got %s", out.Email)
	}
	if out.UserID == uuid.Nil {
		t.Error("user id must not be nil")
	}

	created, ok := repo.users["officer@bank.com"]
	if !ok {
		t.Fatal("officer not found in repository")
	}
	if created.Role != roles.RoleOfficer {
		t.Errorf("want role %s, got %s", roles.RoleOfficer, created.Role)
	}
}
