// Package jwks — загрузка и кэширование JWKS от auth-service.
package jwks

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxJWKSBytes = 1 << 20

type Fetcher interface {
	Fetch(ctx context.Context) (map[string]ed25519.PublicKey, error)
}

type HTTPFetcher struct {
	url    string
	client *http.Client
}

func NewHTTPFetcher(url string, timeout time.Duration) *HTTPFetcher {
	return &HTTPFetcher{url: url, client: &http.Client{Timeout: timeout}}
}

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Use string `json:"use"`
}

// Fetch делает GET {url} и разбирает RFC 7517/8037 JWK Set. Поддерживаются только
// OKP/Ed25519; остальные ключи пропускаются.
func (f *HTTPFetcher) Fetch(ctx context.Context) (map[string]ed25519.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks: unexpected status %d", resp.StatusCode)
	}

	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJWKSBytes)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]ed25519.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" || k.Kid == "" || (k.Use != "" && k.Use != "sig") {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(raw)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("jwks contains no usable Ed25519 keys")
	}
	return keys, nil
}
