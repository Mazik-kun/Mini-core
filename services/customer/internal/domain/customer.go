package domain

import (
	"time"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
)

type Customer struct {
	ID uuid.UUID
	UserID uuid.UUID
	FullName string
	BirthDate pgtype.Date
	Address string
	PhoneNumber string
	Citizenship string
	Status Status
	CreatedAt time.Time
	UpdatedAt time.Time
}