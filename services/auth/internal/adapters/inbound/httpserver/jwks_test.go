package httpserver

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
)

type jwkEntry struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

type jwksResponse struct {
	Keys []jwkEntry `json:"keys"`
}

func TestBuildJWKS(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	const keyID = "test-key-1"

	rawJSON, err := BuildJWKS(publicKey, keyID)
	if err != nil {
		t.Fatalf("BuildJWKS: %v", err)
	}

	var parsed jwksResponse
	if err := json.Unmarshal(rawJSON, &parsed); err != nil {
		t.Fatalf("unmarshal jwks: %v", err)
	}
	if len(parsed.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(parsed.Keys))
	}

	entry := parsed.Keys[0]
	if entry.Kty != "OKP" {
		t.Errorf("kty = %q, want OKP", entry.Kty)
	}
	if entry.Crv != "Ed25519" {
		t.Errorf("crv = %q, want Ed25519", entry.Crv)
	}
	if entry.Alg != "EdDSA" {
		t.Errorf("alg = %q, want EdDSA", entry.Alg)
	}
	if entry.Use != "sig" {
		t.Errorf("use = %q, want sig", entry.Use)
	}
	if entry.Kid != keyID {
		t.Errorf("kid = %q, want %q", entry.Kid, keyID)
	}

	decodedX, err := base64.RawURLEncoding.DecodeString(entry.X)
	if err != nil {
		t.Fatalf("decode x: %v", err)
	}
	if !bytes.Equal(decodedX, publicKey) {
		t.Errorf("x does not match public key")
	}
}

func TestBuildJWKSRejectsEmptyKeyID(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	if _, err := BuildJWKS(publicKey, ""); err == nil {
		t.Fatal("expected error for empty keyID")
	}
}