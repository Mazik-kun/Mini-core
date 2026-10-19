package jwt

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/auth/internal/core/domain"
)

type Verifier struct {
	publicKey ed25519.PublicKey
}

func NewVerifier(publicKeyPath string) (*Verifier, error) {
	pemBytes, err := os.ReadFile(publicKeyPath)
	if err != nil { return nil, fmt.Errorf("jwt: read public key file: %w", err) }
	key, err := LoadPublicKey(pemBytes)
	if err != nil { return nil, err }
	return &Verifier{publicKey: key}, nil
}

func (v *Verifier) Verify(tokenString string) (domain.Claims, error) {
	parsed, err := jwtlib.Parse(tokenString, v.parseToken, jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}))
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return domain.Claims{}, domain.ErrTokenExpired
		}
		return domain.Claims{}, domain.ErrInvalidToken
	}
	if !parsed.Valid { return domain.Claims{}, domain.ErrInvalidToken }

	claims, ok := parsed.Claims.(jwtlib.MapClaims)
	if !ok { return domain.Claims{}, domain.ErrInvalidToken }

	subject, err := claims.GetSubject()
	if err != nil { return domain.Claims{}, domain.ErrInvalidToken }
	userID, err := uuid.Parse(subject)
	if err != nil { return domain.Claims{}, domain.ErrInvalidToken }

	role, _ := claims["role"].(string)
	if role == "" { return domain.Claims{}, domain.ErrInvalidToken }

	var customerID *uuid.UUID
	if raw, ok := claims["customer_id"].(string); ok && raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil { return domain.Claims{}, domain.ErrInvalidToken }
		customerID = &id
	}

	return domain.Claims{
		UserID:     userID,
		Role:       role,
		CustomerID: customerID,
	}, nil
}

// Вынесенная именованная функция вместо анонимной
func (v *Verifier) parseToken(token *jwtlib.Token) (any, error) {
	if _, ok := token.Method.(*jwtlib.SigningMethodEd25519); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	return v.publicKey, nil
}