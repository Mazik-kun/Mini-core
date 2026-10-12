package jwt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type Issuer struct {
	privateKey ed25519.PrivateKey
	accessTTL  time.Duration
}

func NewIssuer(privateKeyPEM []byte, accessTTL time.Duration) (*Issuer, error) {
	if accessTTL <= 0 {
		return nil, errors.New("jwt: accessTTL must be positive")
	}
	key, err := LoadPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	return &Issuer{privateKey: key, accessTTL: accessTTL}, nil
}

func (i *Issuer) Issue(c domain.Claims) (domain.TokenPair, error) {
	now := time.Now().UTC()
	accessToken, err := i.issueAccess(c, now, now.Add(i.accessTTL))
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	return domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(i.accessTTL.Seconds()),
	}, nil
}

func (i *Issuer) issueAccess(c domain.Claims, issuedAt, expiresAt time.Time) (string, error) {
	mapClaims := jwtlib.MapClaims{
		"sub":  c.UserID.String(),
		"role": c.Role,
		"iat":  issuedAt.Unix(),
		"exp":  expiresAt.Unix(),
	}
	if c.CustomerID != nil {
		mapClaims["customer_id"] = c.CustomerID.String()
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, mapClaims)
	return token.SignedString(i.privateKey)
}

func generateRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}