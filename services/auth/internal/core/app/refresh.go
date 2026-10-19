package app

import (
    "context"
    "fmt"
    "time"

    "github.com/Mazik-kun/mini-core/services/auth/internal/core/domain"
)

type RefreshUseCase struct {
    refreshRepo domain.RefreshTokenRepository
    users       domain.UserRepository
    tokens      domain.TokenIssuer
    refreshTTL  time.Duration
}

func NewRefreshUseCase(
    refreshRepo domain.RefreshTokenRepository,
    users domain.UserRepository,
    tokens domain.TokenIssuer,
    refreshTTL time.Duration,
) *RefreshUseCase {
    return &RefreshUseCase{
        refreshRepo: refreshRepo,
        users:       users,
        tokens:      tokens,
        refreshTTL:  refreshTTL,
    }
}

type RefreshInput struct {
    RefreshToken string
}

type RefreshOutput struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int64
}

func (uc *RefreshUseCase) Refresh(ctx context.Context, in RefreshInput) (RefreshOutput, error) {
    if in.RefreshToken == "" {
        return RefreshOutput{}, domain.ErrInvalidToken
    }

    hash := domain.HashRefreshToken(in.RefreshToken)

    stored, err := uc.refreshRepo.GetByHash(ctx, hash)
    if err != nil {
        return RefreshOutput{}, err
    }

    if stored.RevokedAt != nil {
        return RefreshOutput{}, domain.ErrInvalidToken
    }
    if time.Now().UTC().After(stored.ExpiresAt) {
        return RefreshOutput{}, domain.ErrTokenExpired
    }

    // Ротация: старый refresh отзываем
    if err := uc.refreshRepo.Revoke(ctx, stored.ID); err != nil {
        return RefreshOutput{}, fmt.Errorf("revoke old refresh: %w", err)
    }

    // Нужны role и customer_id для новых claims
    user, err := uc.users.GetByID(ctx, stored.UserID)
    if err != nil {
        return RefreshOutput{}, err
    }

    pair, err := uc.tokens.Issue(domain.Claims{
        UserID:     user.ID,
        Role:       user.Role,
        CustomerID: user.CustomerID,
    })
    if err != nil {
        return RefreshOutput{}, fmt.Errorf("issue tokens: %w", err)
    }

    newHash := domain.HashRefreshToken(pair.RefreshToken)
    err = uc.refreshRepo.Create(ctx, domain.RefreshToken{
        UserID:    user.ID,
        TokenHash: newHash,
        ExpiresAt: time.Now().UTC().Add(uc.refreshTTL),
    })
    if err != nil {
        return RefreshOutput{}, fmt.Errorf("save new refresh: %w", err)
    }

    return RefreshOutput{
        AccessToken:  pair.AccessToken,
        RefreshToken: pair.RefreshToken,
        ExpiresIn:    pair.ExpiresIn,
    }, nil
}
