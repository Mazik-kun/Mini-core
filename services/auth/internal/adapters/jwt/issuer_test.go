package jwt

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

func TestIssueVerifyRoundTrip(t *testing.T) {
	priv, err := os.ReadFile("../../../../secrets/jwt_private.pem")
	if err != nil {
		t.Skip("no private key, skip")
	}
	pub, err := os.ReadFile("../../../../secrets/jwt_public.pem")
	if err != nil {
		t.Fatal(err)
	}

	issuer, err := NewIssuer(priv, 15*time.Minute)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	verifier, err := NewVerifier(pub)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}

	custID := uuid.New()
	pair, err := issuer.Issue(domain.Claims{
		UserID:     uuid.New(),
		Role:       "CLIENT",
		CustomerID: &custID,
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	claims, err := verifier.Verify(pair.AccessToken)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.CustomerID == nil || *claims.CustomerID != custID {
		t.Fatal("customer_id mismatch")
	}
}

func TestVerifyExpired(t *testing.T) {
	priv, _ := os.ReadFile("../../../../secrets/jwt_private.pem")
	pub, _ := os.ReadFile("../../../../secrets/jwt_public.pem")
	if priv == nil || pub == nil {
		t.Skip("no keys")
	}
	issuer, _ := NewIssuer(priv, -1*time.Minute) // уже истёк
	verifier, _ := NewVerifier(pub)
	pair, _ := issuer.Issue(domain.Claims{UserID: uuid.New(), Role: "CLIENT"})
	_, err := verifier.Verify(pair.AccessToken)
	if err != domain.ErrTokenExpired {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}
}