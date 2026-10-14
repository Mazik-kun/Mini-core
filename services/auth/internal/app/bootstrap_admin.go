package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

// BootstrapAdmin создаёт первого ADMIN, если в БД ещё нет ни одного.
// Если ADMIN уже есть — ничего не делает.
func BootstrapAdmin(
	ctx context.Context,
	users domain.UserRepository,
	hasher domain.PasswordHasher,
	email string,
	password string,
) error {
	adminExists, err := users.ExistsByRole(ctx, roles.RoleAdmin)
	if err != nil {
		return fmt.Errorf("check admin exists: %w", err)
	}
	if adminExists {
		return nil
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if err := validateEmail(email); err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}

	hash, err := hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	admin, err := users.Create(ctx, domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         roles.RoleAdmin,
	})
	if err != nil {
		return err
	}

	slog.Default().Info("bootstrap admin created",
		"user_id", admin.ID.String(),
		"email", admin.Email,
	)
	return nil
}