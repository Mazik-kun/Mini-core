package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type LoginUseCase struct {
	users       domain.UserRepository
	hasher      domain.PasswordHasher
	tokens      domain.TokenIssuer
	refreshRepo domain.RefreshTokenRepository
	refreshTTL  time.Duration
}

func NewLoginUseCase(
	users domain.UserRepository,
	hasher domain.PasswordHasher,
	tokens domain.TokenIssuer,
	refreshRepo domain.RefreshTokenRepository,
	refreshTTL time.Duration,
) *LoginUseCase {
	return &LoginUseCase{
		users:       users,
		hasher:      hasher,
		tokens:      tokens,
		refreshRepo: refreshRepo,
		refreshTTL:  refreshTTL,
	}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func (uc *LoginUseCase) Login(ctx context.Context, in LoginInput) (LoginOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return LoginOutput{}, domain.ErrInvalidCredentials
		}
		return LoginOutput{}, err
	}

	ok, err := uc.hasher.Verify(in.Password, user.PasswordHash)
	if err != nil {
		return LoginOutput{}, err
	}
	if !ok {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	pair, err := uc.tokens.Issue(domain.Claims{
		UserID:     user.ID,
		Role:       user.Role,
		CustomerID: user.CustomerID,
	})
	if err != nil {
		return LoginOutput{}, err
	}

	err = uc.refreshRepo.Create(ctx, domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: domain.HashRefreshToken(pair.RefreshToken),
		ExpiresAt: time.Now().UTC().Add(uc.refreshTTL),
	})
	if err != nil {
		return LoginOutput{}, err
	}

	return LoginOutput{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}