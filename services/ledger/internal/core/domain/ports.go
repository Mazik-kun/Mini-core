package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type LedgerRepository interface {
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (Transaction, error)
	
	LockBalances(ctx context.Context, tx pgx.Tx, accountIDs []uuid.UUID) (map[uuid.UUID]Balance, error)

	CreateTransaction(ctx context.Context, tx pgx.Tx, t Transaction) error
	CreateEntry(ctx context.Context, tx pgx.Tx, e Entry) error
	UpdateBalance(ctx context.Context, tx pgx.Tx, accountID uuid.UUID, amountChange int64) error

	CreateOutboxEvent(ctx context.Context, tx pgx.Tx, eventType string, payload []byte) error
	GetBalance(ctx context.Context, accountID uuid.UUID) (Balance, error)

	ListTransactions(ctx context.Context, accountID uuid.UUID, fromTime, toTime pgtype.Timestamptz, limit, offset int32) ([]Transaction, error)
}

type AccountClient interface {
	GetAccount(ctx context.Context, accountID uuid.UUID) (currency string, status string, err error)
}
