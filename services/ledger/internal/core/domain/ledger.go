package domain

import (
	"time"
	"github.com/google/uuid"
)

type TransactionType string

const (
	TypeTransfer TransactionType = "TRANSFER"
	TypeDeposit  TransactionType = "DEPOSIT"
)

type Status string
const (
	StatusPending   Status = "PENDING"
	StatusCompleted Status = "COMPLETED"
	StatusDeclined  Status = "DECLINED"
)

type Direction string
const (
	DirectionDebit  Direction = "DEBIT" 
	DirectionCredit Direction = "CREDIT" 
)

type Entry struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	AccountID     uuid.UUID
	Direction     Direction
	Amount        int64 // В копейках/тетри!
	Currency      string
}

type Transaction struct {
	ID             uuid.UUID
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Amount         int64
	Type TransactionType
	Currency       string
	Status         Status
	IdempotencyKey string
	CreatedAt      time.Time
	Entries        []Entry 
}

type Balance struct {
	AccountID uuid.UUID
	Amount    int64
	Currency  string
}