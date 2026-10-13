package authjwt

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	KeyType string `json:"kty"`
	Curve   string `json:"crv"`
	KeyID   string `json:"kid"`
	X       string `json:"x"`
}

// LoadPublicKeyFromFile читает PEM-файл с публичным ключом Ed25519.
func LoadPublicKeyFromFile(path string) (ed25519.PublicKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("authjwt: read public key file %q: %w", path, err)
	}
	return loadPublicKeyFromPEM(pemBytes)
}

// LoadPublicKeyFromJWKS забирает публичный ключ Ed25519 из JWKS по keyID.
func LoadPublicKeyFromJWKS(jwksURL string, keyID string) (ed25519.PublicKey, error) {
	httpClient := http.Client{Timeout: 10 * time.Second}

	response, err := httpClient.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("authjwt: fetch JWKS: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authjwt: JWKS returned status %d", response.StatusCode)
	}

	var document jwksDocument
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		return nil, fmt.Errorf("authjwt: decode JWKS: %w", err)
	}

	for _, key := range document.Keys {
		if key.KeyType != "OKP" || key.Curve != "Ed25519" {
			continue
		}
		if keyID != "" && key.KeyID != keyID {
			continue
		}

		decodedKey, err := base64.RawURLEncoding.DecodeString(key.X)
		if err != nil {
			return nil, fmt.Errorf("authjwt: decode JWKS key %q: %w", key.KeyID, err)
		}
		if len(decodedKey) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("authjwt: JWKS key %q has invalid length %d", key.KeyID, len(decodedKey))
		}

		return ed25519.PublicKey(decodedKey), nil
	}

	return nil, fmt.Errorf("authjwt: no Ed25519 key found for key ID %q", keyID)
}

func loadPublicKeyFromPEM(pemBytes []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("authjwt: failed to decode public key PEM")
	}
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("authjwt: unsupported public key block type %q", block.Type)
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("authjwt: parse PKIX public key: %w", err)
	}

	key, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("authjwt: public key is not Ed25519")
	}
	return key, nil
}