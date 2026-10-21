package httpserver

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"

)

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

func BuildJWKS(publicKey ed25519.PublicKey, keyID string) ([]byte, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("jwks: invalid Ed25519 public key size")
	}
	if keyID == "" {
		return nil, errors.New("jwks: keyID must not be empty")
	}

	doc := jwksDocument{
		Keys: []jwk{{
			Kty: "OKP",
			Crv: "Ed25519",
			X:   base64.RawURLEncoding.EncodeToString(publicKey),
			Kid: keyID,
			Use: "sig",
			Alg: "EdDSA",
		}},
	}
	return json.Marshal(doc)
}