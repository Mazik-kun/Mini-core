package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/postgres/db"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type RefreshTokenRepository struct {
	q *db.Queries
}

func NewRefreshTokenRepository(q *db.Queries) *RefreshTokenRepository {
	return &RefreshTokenRepository{q: q}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, rt domain.RefreshToken) error {
	if rt.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate refresh token id: %w", err)
		}
		rt.ID = id
	}
	_, err := r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:        rt.ID,
		UserID:    rt.UserID,
		TokenHash: rt.TokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: rt.ExpiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	row, err := r.q.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RefreshToken{}, domain.ErrRefreshTokenNotFound
		}
		return domain.RefreshToken{}, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return toRefreshTokenDomain(row), nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	if err := r.q.Revoke(ctx, id); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("revoke all user refresh tokens: %w", err)
	}
	return nil
}

func toRefreshTokenDomain(rt db.RefreshToken) domain.RefreshToken {
	var revokedAt *time.Time
	if rt.RevokedAt.Valid {
		t := rt.RevokedAt.Time
		revokedAt = &t
	}
	return domain.RefreshToken{
		ID:        rt.ID,
		UserID:    rt.UserID,
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt.Time,
		RevokedAt: revokedAt,
		CreatedAt: rt.CreatedAt.Time,
	}
}