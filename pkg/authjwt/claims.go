package authjwt

import (
	"time"

	"github.com/google/uuid"
)

// Claims — набор данных, которые мы достаём из проверенного JWT.
type Claims struct {
	Subject    uuid.UUID
	Role       string
	CustomerID *uuid.UUID
	ExpiresAt  time.Time
	IssuedAt   time.Time
	Issuer     string
}