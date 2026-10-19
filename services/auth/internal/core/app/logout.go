package app

import (
    "context"
    "errors"

    "github.com/Mazik-kun/mini-core/services/auth/internal/core/domain"
)

type LogoutUseCase struct {
    refreshRepo domain.RefreshTokenRepository
}

func NewLogoutUseCase(refreshRepo domain.RefreshTokenRepository) *LogoutUseCase {
    return &LogoutUseCase{refreshRepo: refreshRepo}
}

type LogoutInput struct {
    RefreshToken string
}

func (uc *LogoutUseCase) Logout(ctx context.Context, in LogoutInput) error {
    if in.RefreshToken == "" {
        return nil // идемпотентно: нечего отзывать
    }

    hash := domain.HashRefreshToken(in.RefreshToken)

    stored, err := uc.refreshRepo.GetByHash(ctx, hash)
    if err != nil {
        // Если токена нет — считаем logout успешным (идемпотентность)
        if errors.Is(err, domain.ErrRefreshTokenNotFound) {
            return nil
        }
        return err
    }

    if stored.RevokedAt != nil {
        return nil // уже отозван
    }

    return uc.refreshRepo.Revoke(ctx, stored.ID)
}
