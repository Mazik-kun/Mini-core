package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	CustomerID   *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const RoleClient = "CLIENT"