package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Mazik-kun/mini-core/services/customer/internal/adapters/postgres/db"
	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

const pgUniqueViolation = "23505"

type CustomerRepository struct {
	q *db.Queries
}

func NewCustomerRepository(q *db.Queries) *CustomerRepository {
	return &CustomerRepository{q: q}
}

// ---------- Create ----------

func (r *CustomerRepository) Create(ctx context.Context, cus domain.Customer) (domain.Customer, error) {
	row, err := r.q.CreateCustomer(ctx, db.CreateCustomerParams{
		UserID:      cus.UserID,
		FullName:    cus.FullName,
		BirthDate:   cus.BirthDate,
		Address:     cus.Address,
		PhoneNumber: cus.PhoneNumber,
		Citizenship: cus.Citizenship,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			// В customers уникален user_id → значит, профиль уже создан.
			return domain.Customer{}, domain.ErrAlreadyExists
		}
		return domain.Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return toDomain(row), nil
}

// ---------- GetByID ----------

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Customer, error) {
	row, err := r.q.GetCustomerByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Customer{}, domain.ErrCustomerNotFound
		}
		return domain.Customer{}, fmt.Errorf("get customer by id: %w", err)
	}
	return toDomain(row), nil
}

// ---------- GetByUserID ----------

func (r *CustomerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (domain.Customer, error) {
	row, err := r.q.GetCustomerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Customer{}, domain.ErrCustomerNotFound
		}
		return domain.Customer{}, fmt.Errorf("get customer by user_id: %w", err)
	}
	return toDomain(row), nil
}

// ---------- Update ----------

func (r *CustomerRepository) Update(ctx context.Context, cus domain.Customer) (domain.Customer, error) {
	row, err := r.q.UpdateCustomer(ctx, db.UpdateCustomerParams{
		ID:          cus.ID,
		FullName:    cus.FullName,
		BirthDate:   cus.BirthDate,
		Address:     cus.Address,
		PhoneNumber: cus.PhoneNumber,
		Citizenship: cus.Citizenship,
		Status:      string(cus.Status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Customer{}, domain.ErrCustomerNotFound
		}
		return domain.Customer{}, fmt.Errorf("update customer: %w", err)
	}
	return toDomain(row), nil
}

// ---------- UpdateStatus ----------

func (r *CustomerRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status) error {
	_, err := r.q.UpdateCustomerStatus(ctx, db.UpdateCustomerStatusParams{
		ID:     id,
		Status: string(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrCustomerNotFound
		}
		return fmt.Errorf("update customer status: %w", err)
	}
	return nil
}

// ---------- List ----------
//
// В SQL используется:
//   - @statuses::text[] IS NULL  → «без фильтра»
//   - sqlc.narg('cursor')::uuid  → optional-параметр
//   - LIMIT @lim
//
// По этой причине:
//   - пустой срез Statuses конвертируем в nil, чтобы сработало "IS NULL";
//   - курсор кладём в pgtype.UUID с Valid=true только если он не uuid.Nil.
//
// TODO: сверить имена полей в db.ListCustomersParams (Statuses/Cursor/Lim)
//       и точный тип Cursor (pgtype.UUID vs *uuid.UUID) с генерированным db/*.go.
func (r *CustomerRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Customer, error) {
	params := db.ListCustomersParams{
		Lim: int32(filter.Limit),
	}

	// statuses: []domain.Status → []string | nil
	if len(filter.Statuses) > 0 {
		statuses := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			statuses[i] = string(s)
		}
		params.Statuses = statuses
	}

	// cursor: uuid.UUID → pgtype.UUID (Valid=false → NULL)
	if filter.Cursor != uuid.Nil {
		params.Cursor = pgtype.UUID{
			Bytes: filter.Cursor,
			Valid: true,
		}
	}

	rows, err := r.q.ListCustomers(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	out := make([]domain.Customer, len(rows))
	for i, row := range rows {
		out[i] = toDomain(row)
	}
	return out, nil
}

// ---------- mapper ----------

func toDomain(cus db.Customer) domain.Customer {
	return domain.Customer{
		ID:          cus.ID,
		UserID:      cus.UserID,
		FullName:    cus.FullName,
		BirthDate:   cus.BirthDate, // в domain.Customer поле тоже pgtype.Date
		Address:     cus.Address,
		PhoneNumber: cus.PhoneNumber,
		Citizenship: cus.Citizenship,
		Status:      domain.Status(cus.Status),
		CreatedAt:   cus.CreatedAt.Time,
		UpdatedAt:   cus.UpdatedAt.Time,
	}
}