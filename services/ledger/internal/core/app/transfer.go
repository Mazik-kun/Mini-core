package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransferUseCase struct {
	db *pgxpool.Pool
	repo domain.LedgerRepository
	accountClient domain.AccountClient
}

func NewTransferUseCase(db *pgxpool.Pool, repo domain.LedgerRepository, accountClient domain.AccountClient) *TransferUseCase{
	return &TransferUseCase{db:db, repo:repo, accountClient: accountClient}
}

type TransferInput struct{
	FromAccountID uuid.UUID
	ToAccountID uuid.UUID
	Amount int64
	Currency string
	IdempotencyKey string
}

func(uc *TransferUseCase)Transfer(ctx context.Context, in TransferInput)(domain.Transaction, error){
	if in.Amount <= 0{
		return domain.Transaction{}, fmt.Errorf("amount must be positive")
	}
	if in.FromAccountID == in.ToAccountID{
		return domain.Transaction{}, domain.ErrSameAccount
	}

	existing, err := uc.repo.GetTransactionByIdempotencyKey(ctx, in.IdempotencyKey)

	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows){
		return domain.Transaction{}, fmt.Errorf("check idempotency: %w", err)
	}
	// 1. Проверяем счет отправителя
	currencyFrom, statusFrom, err := uc.accountClient.GetAccount(ctx, in.FromAccountID)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("get from account: %w", err)
	}
	if statusFrom != "ACCOUNT_STATUS_ACTIVE" {
		return domain.Transaction{}, fmt.Errorf("from account is not active")
	}

	// 2. Проверяем счет получателя
	currencyTo, statusTo, err := uc.accountClient.GetAccount(ctx, in.ToAccountID)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("get to account: %w", err)
	}
	if statusTo != "ACCOUNT_STATUS_ACTIVE" {
		return domain.Transaction{}, fmt.Errorf("to account is not active")
	}

	// 3. Проверяем, что валюты совпадают
	if currencyFrom != currencyTo {
		return domain.Transaction{}, domain.ErrCurrencyMismatch
	}
	if currencyFrom != in.Currency {
		return domain.Transaction{}, fmt.Errorf("currency mismatch with request")
	}

	tx, err := uc.db.Begin(ctx)
	if err != nil{
		return domain.Transaction{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	accIDs := []uuid.UUID{in.FromAccountID, in.ToAccountID}

	balances, err := uc.repo.LockBalances(ctx, tx, accIDs)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("lock balances: %w", err)
	}

	fromBalance, ok := balances[in.FromAccountID]
	if !ok{
		return domain.Transaction{}, fmt.Errorf("from account balance not found")
	}

	if fromBalance.Amount < in.Amount{
		return domain.Transaction{}, domain.ErrInsufficientFunds
	}
	txID := uuid.New()
	transaction := domain.Transaction{
		ID: txID,
		FromAccountID: in.FromAccountID,
		ToAccountID: in.ToAccountID,
		Amount: in.Amount,
		Currency: in.Currency,
		Status: domain.StatusCompleted,
		Type: domain.TypeTransfer,
		IdempotencyKey: in.IdempotencyKey,
		CreatedAt: time.Now(),
	}

	if err := uc.repo.CreateTransaction(ctx, tx, transaction); err != nil{
		return domain.Transaction{}, fmt.Errorf("create transaction: %w", err)
	}

	debitEntry := domain.Entry{
		ID: uuid.New(), TransactionID: txID, AccountID: in.FromAccountID,
		Direction: domain.DirectionDebit, Amount: in.Amount, Currency: in.Currency,
	}

	creditEntry := domain.Entry{
		ID: uuid.New(), TransactionID: txID, AccountID: in.ToAccountID,
		Direction: domain.DirectionCredit, Amount: in.Amount, Currency: in.Currency,
	}

	if err := uc.repo.CreateEntry(ctx, tx, debitEntry); err != nil{
		return domain.Transaction{}, err
	}
	if err := uc.repo.CreateEntry(ctx, tx, creditEntry); err != nil{
		return domain.Transaction{}, err
	}

	if err:= uc.repo.UpdateBalance(ctx, tx, in.FromAccountID, -in.Amount); err != nil{
		return domain.Transaction{}, err
	}
	if err := uc.repo.UpdateBalance(ctx, tx, in.ToAccountID, in.Amount); err != nil{
		return domain.Transaction{}, err
	}
	
	eventData, _ := json.Marshal(map[string]interface{}{"transaction_id": txID, "status": "COMPLETED"})
	if err := uc.repo.CreateOutboxEvent(ctx, tx, "ledger.transaction_completed", eventData); err != nil {
		return domain.Transaction{}, fmt.Errorf("outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil{
		return domain.Transaction{}, fmt.Errorf("commit tx: %w", err)
	}
	transaction.Entries = []domain.Entry{debitEntry, creditEntry}
	return transaction, nil
}