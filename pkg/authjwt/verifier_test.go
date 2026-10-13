package authjwt

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestVerifyToken(t *testing.T) {
	publicKey, privateKey := generateEd25519KeyPair(t)

	userID := uuid.New()
	customerID := uuid.New()

	tokenString := signToken(t, privateKey, jwtlib.MapClaims{
		"sub":         userID.String(),
		"role":        "CLIENT",
		"customer_id": customerID.String(),
		"iat":         time.Now().UTC().Unix(),
		"exp":         time.Now().UTC().Add(time.Hour).Unix(),
		"iss":         "auth-service",
	})

	verifier := NewVerifier(publicKey)

	claims, err := verifier.VerifyToken(tokenString)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}

	if claims.Subject != userID {
		t.Fatalf("Subject = %v, want %v", claims.Subject, userID)
	}
	if claims.Role != "CLIENT" {
		t.Fatalf("Role = %q, want %q", claims.Role, "CLIENT")
	}
	if claims.CustomerID == nil || *claims.CustomerID != customerID {
		t.Fatalf("CustomerID = %v, want %v", claims.CustomerID, customerID)
	}
	if claims.Issuer != "auth-service" {
		t.Fatalf("Issuer = %q, want %q", claims.Issuer, "auth-service")
	}
}

func TestVerifyTokenExpired(t *testing.T) {
	publicKey, privateKey := generateEd25519KeyPair(t)

	tokenString := signToken(t, privateKey, jwtlib.MapClaims{
		"sub":  uuid.New().String(),
		"role": "CLIENT",
		"iat":  time.Now().UTC().Add(-2 * time.Hour).Unix(),
		"exp":  time.Now().UTC().Add(-time.Hour).Unix(),
	})

	verifier := NewVerifier(publicKey)

	_, err := verifier.VerifyToken(tokenString)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("VerifyToken() error = %v, want ErrTokenExpired", err)
	}
}

func TestVerifyTokenWrongKey(t *testing.T) {
	publicKey, _ := generateEd25519KeyPair(t)
	_, wrongPrivateKey := generateEd25519KeyPair(t)

	tokenString := signToken(t, wrongPrivateKey, jwtlib.MapClaims{
		"sub":  uuid.New().String(),
		"role": "CLIENT",
		"iat":  time.Now().UTC().Unix(),
		"exp":  time.Now().UTC().Add(time.Hour).Unix(),
	})

	verifier := NewVerifier(publicKey)

	_, err := verifier.VerifyToken(tokenString)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("VerifyToken() error = %v, want ErrTokenInvalid", err)
	}
}

func TestContextHelpers(t *testing.T) {
	userID := uuid.New()
	claims := Claims{
		Subject: userID,
		Role:    "CLIENT",
	}

	ctx := WithClaims(context.Background(), claims)

	gotClaims, err := ClaimsFromContext(ctx)
	if err != nil {
		t.Fatalf("ClaimsFromContext() error = %v", err)
	}
	if gotClaims.Subject != userID {
		t.Fatalf("Subject = %v, want %v", gotClaims.Subject, userID)
	}

	if err := RequireRole(ctx, "CLIENT"); err != nil {
		t.Fatalf("RequireRole() error = %v", err)
	}

	if err := RequireSelfOrRole(ctx, userID, "ADMIN"); err != nil {
		t.Fatalf("RequireSelfOrRole() error = %v", err)
	}
}

func generateEd25519KeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 key pair: %v", err)
	}
	return publicKey, privateKey
}

func signToken(t *testing.T, privateKey ed25519.PrivateKey, claims jwtlib.MapClaims) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenString
}