package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/auth/internal/core/domain"
)

func TestIssueVerifyRoundTrip(t *testing.T) {
	privPath := "../../../../../secrets/jwt_private.pem"
	pubPath := "../../../../../secrets/jwt_public.pem"

	issuer, err := NewIssuer(privPath, 15*time.Minute)
	if err != nil {
		t.Skip("no private key, skip")
	}
	verifier, err := NewVerifier(pubPath)
	if err != nil {
		t.Fatal(err)
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
	privPath := "../../../../../secrets/jwt_private.pem"
	pubPath := "../../../../../secrets/jwt_public.pem"

	issuer, err := NewIssuer(privPath, -1*time.Minute)
	if err != nil {
		t.Skip("no private key, skip")
	}
	verifier, _ := NewVerifier(pubPath)
	
	pair, _ := issuer.Issue(domain.Claims{UserID: uuid.New(), Role: "CLIENT"})
	_, err = verifier.Verify(pair.AccessToken)
	if err != domain.ErrTokenExpired {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}
}