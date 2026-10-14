package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type CreateOfficerUseCase struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
}

func NewCreateOfficerUseCase(users domain.UserRepository, hasher domain.PasswordHasher) *CreateOfficerUseCase {
	return &CreateOfficerUseCase{users: users, hasher: hasher}
}

type CreateOfficerInput struct {
	Email    string
	Password string
}

type CreateOfficerOutput struct {
	UserID uuid.UUID
	Email  string
}

func (uc *CreateOfficerUseCase) CreateOfficer(ctx context.Context, in CreateOfficerInput) (CreateOfficerOutput, error) {
	claims, err := authjwt.ClaimsFromContext(ctx)
	if err != nil {
		return CreateOfficerOutput{}, domain.ErrPermissionDenied
	}
	if claims.Role != roles.RoleAdmin {
		return CreateOfficerOutput{}, domain.ErrPermissionDenied
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	if err := validateEmail(email); err != nil {
		return CreateOfficerOutput{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return CreateOfficerOutput{}, err
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return CreateOfficerOutput{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := uc.users.Create(ctx, domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         roles.RoleOfficer,
	})
	if err != nil {
		return CreateOfficerOutput{}, err
	}

	return CreateOfficerOutput{UserID: user.ID, Email: user.Email}, nil
}