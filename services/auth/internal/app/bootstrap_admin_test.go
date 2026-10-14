package app_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/auth/internal/app"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

func silenceSlog() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestBootstrapAdmin_CreatesAdmin(t *testing.T) {
	silenceSlog()
	repo := &fakeRepo{users: map[string]domain.User{}}

	err := app.BootstrapAdmin(
		context.Background(),
		repo,
		fakeHasher{},
		"admin@bank.com",
		"password123",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	admin, ok := repo.users["admin@bank.com"]
	if !ok {
		t.Fatal("admin not created")
	}
	if admin.Role != roles.RoleAdmin {
		t.Errorf("want role %s, got %s", roles.RoleAdmin, admin.Role)
	}
}

func TestBootstrapAdmin_SkipsWhenAdminExists(t *testing.T) {
	silenceSlog()
	repo := &fakeRepo{users: map[string]domain.User{}}
	repo.users["existing@bank.com"] = domain.User{
		Email: "existing@bank.com",
		Role:  roles.RoleAdmin,
	}

	err := app.BootstrapAdmin(
		context.Background(),
		repo,
		fakeHasher{},
		"new@bank.com",
		"password123",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.users["new@bank.com"]; ok {
		t.Error("second admin must not be created")
	}
}