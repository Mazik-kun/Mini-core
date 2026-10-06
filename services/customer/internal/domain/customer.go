package domain

import (
	"time"
	"uuid"
)

type Customer struct {
	ID uuid.UUID
	UserID uuid.UUID
	FullName string
	BirthDate time.Time
	Address string
	PhoneNumber string
	Citizenship string
	Status Status
	CreatedAt time.Time
	UpdatedAt time.Time
}