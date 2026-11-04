package domain

import (
	"time"
	"github.com/google/uuid"
)

type Status string
const (
	StatusActive  Status = "ACTIVE"
	StatusBlocked Status = "BLOCKED"
	StatusClosed  Status = "CLOSED"
)

type Type string
const (
	TypeChecking Type = "CHECKING"
	TypeSavings  Type = "SAVINGS"
)

type Account struct {
	ID            uuid.UUID
	CustomerID    uuid.UUID
	AccountNumber string
	Currency      string
	Type          Type
	Status        Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
}