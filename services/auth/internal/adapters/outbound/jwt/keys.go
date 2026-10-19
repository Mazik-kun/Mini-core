package jwt

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func LoadPrivateKey(pemBytes []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("jwt: failed to decode private key PEM")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("jwt: unsupported private key block type %q", block.Type)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("jwt: parse PKCS8 private key: %w", err)
	}
	key, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("jwt: private key is not Ed25519")
	}
	return key, nil
}

func LoadPublicKey(pemBytes []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("jwt: failed to decode public key PEM")
	}
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("jwt: unsupported public key block type %q", block.Type)
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("jwt: parse PKIX public key: %w", err)
	}
	key, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("jwt: public key is not Ed25519")
	}
	return key, nil
}
func LoadPublicKeyFromFile(path string) (ed25519.PublicKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key file: %w", err)
	}
	return LoadPublicKey(pemBytes)
}