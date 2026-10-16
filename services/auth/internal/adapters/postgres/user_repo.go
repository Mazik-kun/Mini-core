package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/postgres/db"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

const pgUniqueViolation = "23505"

type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	if u.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return domain.User{}, fmt.Errorf("generate user id: %w", err)
		}
		u.ID = id
	}
	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return domain.User{}, domain.ErrEmailAlreadyExists
		}
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return toDomain(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return toDomain(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return toDomain(row), nil
}

func (r *UserRepository) ExistsByRole(ctx context.Context, role string) (bool, error) {
	exists, err := r.q.ExistsUserByRole(ctx, role)
	if err != nil {
		return false, fmt.Errorf("exists user by role: %w", err)
	}
	return exists, nil
}

func (r *UserRepository) UpdateCustomerID(ctx context.Context, userID, customerID uuid.UUID) error {
	err := r.q.UpdateUserCustomerID(ctx, db.UpdateUserCustomerIDParams{
		ID:         userID,
		CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update user customer_id: %w", err)
	}
	return nil
}

func toDomain(u db.User) domain.User {
	var customerID *uuid.UUID
	if u.CustomerID.Valid {
		id := uuid.UUID(u.CustomerID.Bytes)
		customerID = &id
	}
	return domain.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		CustomerID:   customerID,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}