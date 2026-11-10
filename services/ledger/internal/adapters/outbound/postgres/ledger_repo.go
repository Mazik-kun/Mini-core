package postgres

import (
	"context"
	"fmt"

	"github.com/Mazik-kun/mini-core/services/ledger/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type LedgerRepo struct {
	q *db.Queries 
}
func NewLedgerRepo(q *db.Queries) *LedgerRepo {
	return &LedgerRepo{q: q}
}

func (r *LedgerRepo) ListTransactions(ctx context.Context, accountID uuid.UUID, fromTime, toTime pgtype.Timestamptz, limit, offset int32)([]domain.Transaction, error){
	
	rows, err := r.q.GetTransactionByAccountID(ctx, db.GetTransactionByAccountIDParams{
		AccountID: accountID,
		FromTime: fromTime,
		ToTime: toTime,
		Limit: limit,
		Offset: offset,
	})
	if err != nil{
		return nil, fmt.Errorf("query transaction: %w", err)
	}
	result := make([]domain.Transaction, 0, len(rows))

	for _, row := range rows{

		entriesRows, err := r.q.GetEntriesByTransactionID(ctx, row.ID)
		if err != nil{
			return nil, fmt.Errorf("get entries for tx %s: %w", row.ID, err)
		}

		var fromAcc, toAcc uuid.UUID
		var amount int64
		var currency string
	
		for _, e := range entriesRows {
			if e.Direction == "DEBIT"{
			fromAcc = e.AccountID
			amount = e.Amount
			currency = e.Currency
		} else if e.Direction == "CREDIT"{
			toAcc = e.AccountID
		}
	}
		result = append(result, domain.Transaction{
			ID: row.ID,
			Status: domain.Status(row.Status),
			Type: domain.TransactionType(row.Type),
			IdempotencyKey: row.IdempotencyKey,
			CreatedAt: row.CreatedAt.Time,
			FromAccountID: fromAcc,
			ToAccountID: toAcc,
			Amount: amount,
			Currency: currency,
			})
	}

	return result, nil
}

func (r *LedgerRepo) GetTransactionByIdempotencyKey(ctx context.Context, key string) (domain.Transaction, error) {
	// 1. Получаем саму транзакцию
	row, err := r.q.GetTransactionByIdempotencyKey(ctx, key)
	if err != nil {
		return domain.Transaction{}, err
	}

	entriesRows, err := r.q.GetEntriesByTransactionID(ctx, row.ID)
	if err != nil {
		return domain.Transaction{}, err
	}

	var fromAcc, toAcc uuid.UUID
	var amount int64
	var currency string

	for _, e := range entriesRows {
		switch domain.Direction(e.Direction) {
		case domain.DirectionDebit:
			fromAcc = e.AccountID
			amount = e.Amount
			currency = e.Currency
		case domain.DirectionCredit:
			toAcc = e.AccountID
		}
	}

	return domain.Transaction{
		ID:             row.ID,
		Status:         domain.Status(row.Status),
		Type:           domain.TransactionType(row.Type),
		FromAccountID:  fromAcc,
		ToAccountID:    toAcc,
		Amount:         amount,
		Currency:       currency,
		IdempotencyKey: row.IdempotencyKey,
		CreatedAt:      row.CreatedAt.Time,
	}, nil
}

func (r *LedgerRepo) LockBalances(ctx context.Context, tx pgx.Tx, accountIDs []uuid.UUID) (map[uuid.UUID]domain.Balance, error) {
	qtx := r.q.WithTx(tx)

	rows, err := qtx.GetBalancesForUpdate(ctx, db.GetBalancesForUpdateParams{
		AccountID:   accountIDs[0],
		AccountID_2: accountIDs[1],
	})
	if err != nil {
		return nil, fmt.Errorf("query lock balances: %w", err)
	}

	result := make(map[uuid.UUID]domain.Balance)

	for _, id := range accountIDs {
		result[id] = domain.Balance{
			AccountID: id,
			Amount:    0,
			Currency:  "GEL",
		}
	}

	for _, row := range rows {
		result[row.AccountID] = domain.Balance{
			AccountID: row.AccountID,
			Amount:    row.Amount,
			Currency:  row.Currency,
		}
	}
	return result, nil
}

func (r *LedgerRepo)CreateTransaction(ctx context.Context, tx pgx.Tx, t domain.Transaction) error{
	qtx := r.q.WithTx(tx)

	_, err:= qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		ID: t.ID,
		Status: string(t.Status),
		Type: string(t.Type),
		IdempotencyKey: t.IdempotencyKey,
	})
	if err != nil{
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (r *LedgerRepo) CreateEntry(ctx context.Context, tx pgx.Tx, e domain.Entry) error {
	qtx := r.q.WithTx(tx)

	_, err := qtx.CreateEntry(ctx, db.CreateEntryParams{
		ID:            e.ID,
		TransactionID: e.TransactionID,
		AccountID:     e.AccountID,
		Direction:     string(e.Direction),
		Amount:        e.Amount,
		Currency:      e.Currency,
	})
	if err != nil {
		return fmt.Errorf("create entry: %w", err)
	}
	return nil
}

func (r *LedgerRepo) UpdateBalance(ctx context.Context, tx pgx.Tx, accountID uuid.UUID, amountChange int64) error {
	qtx := r.q.WithTx(tx)

	err := qtx.UpsertBalance(ctx, db.UpsertBalanceParams{
		AccountID: accountID,
		Amount:    amountChange,
		Currency:  "GEL",
	})
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	return nil
}

func (r *LedgerRepo) CreateOutboxEvent(ctx context.Context, tx pgx.Tx, eventType string, payload []byte) error {
	qtx := r.q.WithTx(tx)

	_, err := qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		ID:        uuid.New(),
		EventType: eventType,
		Payload:   payload,
	})
	if err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}
	return nil
}

func (r *LedgerRepo) GetBalance(ctx context.Context, accountID uuid.UUID) (domain.Balance, error) {
	row, err := r.q.GetBalance(ctx, accountID)
	if err != nil {
		return domain.Balance{}, err
	}
	return domain.Balance{
		AccountID: row.AccountID,
		Amount:    row.Amount,
		Currency:  row.Currency,
	}, nil
}