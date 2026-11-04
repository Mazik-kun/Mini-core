package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
	"github.com/Mazik-kun/mini-core/services/account/internal/adapters/outbound/postgres/db" // сгенерированный sqlc
)

type AccountRepo struct {
	q *db.Queries
}

func NewAccountRepo(q *db.Queries) *AccountRepo { return &AccountRepo{q: q} }

func (r *AccountRepo) Create(ctx context.Context, acc domain.Account) (domain.Account, error) {
	res, err := r.q.CreateAccount(ctx, db.CreateAccountParams{
		ID:            acc.ID,
		CustomerID:    acc.CustomerID,
		AccountNumber: acc.AccountNumber,
		Currency:      acc.Currency,
		AccountType:   string(acc.Type),
		Status:        string(acc.Status),
	})
	if err != nil { return domain.Account{}, err }
	return toDomain(res), nil
}

func (r *AccountRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Account, error) {
	res, err := r.q.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return domain.Account{}, domain.ErrAccountNotFound }
		return domain.Account{}, err
	}
	return toDomain(res), nil
}

func (r *AccountRepo) CountByCurrency(ctx context.Context, cid uuid.UUID, cur string) (int64, error) {
	return r.q.CountAccountsByCurrency(ctx, db.CountAccountsByCurrencyParams{CustomerID: cid, Currency: cur})
}

func (r *AccountRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status) (domain.Account, error) {
	res, err := r.q.UpdateAccountStatus(ctx, db.UpdateAccountStatusParams{ID: id, Status: string(status)})
	if err != nil { return domain.Account{}, err }
	return toDomain(res), nil
}
func (r *AccountRepo) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.Account, error) {
	res, err := r.q.ListAccountsByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	accounts := make([]domain.Account, 0, len(res))
	for _, a := range res {
		accounts = append(accounts, toDomain(a))
	}
	return accounts, nil
}

func toDomain(d db.Account) domain.Account {
	return domain.Account{
		ID:            d.ID,
		CustomerID:    d.CustomerID,
		AccountNumber: d.AccountNumber,
		Currency:      d.Currency,
		Type:          domain.Type(d.AccountType),
		Status:        domain.Status(d.Status),
		CreatedAt:     d.CreatedAt.Time,
		UpdatedAt:     d.UpdatedAt.Time,
	}
}