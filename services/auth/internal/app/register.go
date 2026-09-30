package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type RegisterUseCase struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
}

func NewRegisterUseCase(users domain.UserRepository, hasher domain.PasswordHasher) *RegisterUseCase {
	return &RegisterUseCase{users: users, hasher: hasher}
}

type RegisterInput struct {
	Email    string
	Password string
}

type RegisterOutput struct {
	UserID uuid.UUID
	Email  string
}

func (uc *RegisterUseCase) Execute(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if err := validateEmail(email); err != nil {
		return RegisterOutput{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return RegisterOutput{}, err
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	created, err := uc.users.Create(ctx, domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         domain.RoleClient,
	})
	if err != nil {
		return RegisterOutput{}, err
	}

	return RegisterOutput{UserID: created.ID, Email: created.Email}, nil
}