package authjwt

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func signed(t *testing.T, priv ed25519.PrivateKey, kid string, exp time.Time) string {
	t.Helper()
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, jwtlib.MapClaims{
		"sub": uuid.NewString(), "role": "CLIENT", "exp": exp.Unix(),
	})
	if kid != "" {
		tok.Header["kid"] = kid
	}
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestKeyID(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	if kid, err := KeyID(signed(t, priv, "auth-1", time.Now().Add(time.Minute))); err != nil || kid != "auth-1" {
		t.Fatalf("kid=%q err=%v", kid, err)
	}
	if _, err := KeyID(signed(t, priv, "", time.Now().Add(time.Minute))); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("missing kid must be invalid, got %v", err)
	}
	if _, err := KeyID("garbage"); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("garbage must be invalid, got %v", err)
	}
}

func TestVerifyWithKey(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := VerifyWithKey(signed(t, priv, "k", time.Now().Add(time.Minute)), pub); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyWithKey(signed(t, priv, "k", time.Now().Add(-time.Minute)), pub); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := VerifyWithKey(signed(t, priv, "k", time.Now().Add(time.Minute)), otherPub); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("want ErrTokenInvalid, got %v", err)
	}
}
