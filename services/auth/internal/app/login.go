package app

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type LoginUseCase struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
}

func NewLoginUseCase(users domain.UserRepository, hasher domain.PasswordHasher) *LoginUseCase {
	return &LoginUseCase{users: users, hasher: hasher}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	UserID uuid.UUID
	Role   string
}

func (uc *LoginUseCase) Login(ctx context.Context, in LoginInput) (LoginOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return LoginOutput{}, err
	}

	ok, err := uc.hasher.Verify(in.Password, user.PasswordHash)
	if err != nil {
		return LoginOutput{}, err
	}
	if !ok {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	return LoginOutput{UserID: user.ID, Role: user.Role}, nil
}