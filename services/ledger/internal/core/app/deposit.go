package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
)

// Системный счет для кассы (можно создать его в БД отдельно, но для тестов сгодится так)
var SystemCashAccountID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type DepositUseCase struct {
	db         *pgxpool.Pool
	repo       domain.LedgerRepository
	accountClient domain.AccountClient
}

func NewDepositUseCase(db *pgxpool.Pool, repo domain.LedgerRepository, accountClient domain.AccountClient) *DepositUseCase {
	return &DepositUseCase{db: db, repo: repo, accountClient: accountClient}
}

type DepositInput struct {
	AccountID      uuid.UUID
	Amount         int64
	Currency       string
	IdempotencyKey string
}

func (uc *DepositUseCase) Deposit(ctx context.Context, in DepositInput) (domain.Transaction, error) {
	if in.Amount <= 0 {
		return domain.Transaction{}, fmt.Errorf("amount must be positive")
	}

	// 1. Идемпотентность
	existing, err := uc.repo.GetTransactionByIdempotencyKey(ctx, in.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, fmt.Errorf("check idempotency: %w", err)
	}

	// 2. Проверка счета
	_, status, err := uc.accountClient.GetAccount(ctx, in.AccountID)
	if err != nil || status != "ACCOUNT_STATUS_ACTIVE" {
		return domain.Transaction{}, fmt.Errorf("account is not active")
	}

	// 3. Начинаем транзакцию
	tx, err := uc.db.Begin(ctx)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 4. Блокируем баланс клиента (кассу можно не блокировать, так как туда только пишем)
	accIDs := []uuid.UUID{in.AccountID, SystemCashAccountID}
	_, err = uc.repo.LockBalances(ctx, tx, accIDs) // Для депозита нам не нужно проверять, сколько там денег, просто лочим
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("lock balance: %w", err)
	}

	// 5. Создаем транзакцию
	txID := uuid.New()
	transaction := domain.Transaction{
		ID:             txID,
		FromAccountID:  SystemCashAccountID, // Списываем с кассы
		ToAccountID:    in.AccountID,        // Начисляем клиенту
		Amount:         in.Amount,
		Currency:       in.Currency,
		Status:         domain.StatusCompleted,
		Type:           domain.TypeDeposit,
		IdempotencyKey: in.IdempotencyKey,
		CreatedAt:      time.Now(),
	}

	if err := uc.repo.CreateTransaction(ctx, tx, transaction); err != nil {
		return domain.Transaction{}, fmt.Errorf("create transaction: %w", err)
	}

	// 6. Проводки
	debitEntry := domain.Entry{
		ID: uuid.New(), TransactionID: txID, AccountID: SystemCashAccountID,
		Direction: domain.DirectionDebit, Amount: in.Amount, Currency: in.Currency,
	}
	creditEntry := domain.Entry{
		ID: uuid.New(), TransactionID: txID, AccountID: in.AccountID,
		Direction: domain.DirectionCredit, Amount: in.Amount, Currency: in.Currency,
	}
	if err := uc.repo.CreateEntry(ctx, tx, debitEntry); err != nil { return domain.Transaction{}, err }
	if err := uc.repo.CreateEntry(ctx, tx, creditEntry); err != nil { return domain.Transaction{}, err }

	// 7. Обновляем балансы
	// Касса уходит в минус (это нормально для системного счета)
	if err := uc.repo.UpdateBalance(ctx, tx, SystemCashAccountID, -in.Amount); err != nil { return domain.Transaction{}, err }
	if err := uc.repo.UpdateBalance(ctx, tx, in.AccountID, in.Amount); err != nil { return domain.Transaction{}, err }

	// 8. Outbox
	eventData, _ := json.Marshal(map[string]interface{}{
		"transaction_id": txID,
		"status":         "COMPLETED",
		"amount":         in.Amount,
	})
	if err := uc.repo.CreateOutboxEvent(ctx, tx, "ledger.transaction_completed", eventData); err != nil {
		return domain.Transaction{}, fmt.Errorf("outbox: %w", err)
	}

	// 9. Commit
	if err := tx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit tx: %w", err)
	}

	return transaction, nil
}