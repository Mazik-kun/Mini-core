package authjwt

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenExpired     = errors.New("authjwt: token expired")
	ErrTokenInvalid     = errors.New("authjwt: token invalid")
	ErrPermissionDenied = errors.New("authjwt: permission denied")
)

type Verifier struct {
	publicKey ed25519.PublicKey
}

func NewVerifier(publicKey ed25519.PublicKey) *Verifier {
	return &Verifier{publicKey: publicKey}
}

func (v *Verifier) VerifyToken(tokenString string) (Claims, error) {
	parsedToken, err := jwtlib.Parse(
		tokenString,
		v.verificationKey,
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return Claims{}, ErrTokenExpired
		}
		return Claims{}, ErrTokenInvalid
	}

	if !parsedToken.Valid {
		return Claims{}, ErrTokenInvalid
	}

	mapClaims, ok := parsedToken.Claims.(jwtlib.MapClaims)
	if !ok {
		return Claims{}, ErrTokenInvalid
	}

	return mapClaimsToClaims(mapClaims)
}

func (v *Verifier) verificationKey(token *jwtlib.Token) (any, error) {
	if _, ok := token.Method.(*jwtlib.SigningMethodEd25519); !ok {
		return nil, fmt.Errorf("authjwt: unexpected signing method: %v", token.Header["alg"])
	}
	return v.publicKey, nil
}

func mapClaimsToClaims(mapClaims jwtlib.MapClaims) (Claims, error) {
	subject, err := mapClaims.GetSubject()
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}

	role, _ := mapClaims["role"].(string)
	if role == "" {
		return Claims{}, ErrTokenInvalid
	}

	var customerID *uuid.UUID
	if rawCustomerID, ok := mapClaims["customer_id"].(string); ok && rawCustomerID != "" {
		parsedCustomerID, err := uuid.Parse(rawCustomerID)
		if err != nil {
			return Claims{}, ErrTokenInvalid
		}
		customerID = &parsedCustomerID
	}

	expiresAt, err := mapClaims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return Claims{}, ErrTokenInvalid
	}

	issuedAt, _ := mapClaims.GetIssuedAt()
	issuer, _ := mapClaims.GetIssuer()

	claims := Claims{
		Subject:    userID,
		Role:       role,
		CustomerID: customerID,
		ExpiresAt:  expiresAt.Time,
		Issuer:     issuer,
	}
	if issuedAt != nil {
		claims.IssuedAt = issuedAt.Time
	}

	return claims, nil
}

var _ = time.Time{}