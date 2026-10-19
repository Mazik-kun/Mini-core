package domain

import (
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID
	Role string
	CustomerID *uuid.UUID
}

type TokenPair struct {
	AccessToken string
	RefreshToken string
	ExpiresIn int64
}

type TokenIssuer interface {
	Issue(c Claims) (TokenPair, error)
}

type TokenVerifier interface{
	Verify(token string) (Claims, error)
}