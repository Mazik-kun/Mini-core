package middleware

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

type fakeJWKS struct {
	keys map[string]ed25519.PublicKey
	err  error
}

func (f fakeJWKS) KeyByID(_ context.Context, kid string) (ed25519.PublicKey, error) {
	if f.err != nil {
		return nil, f.err
	}
	if k, ok := f.keys[kid]; ok {
		return k, nil
	}
	return nil, port.ErrKeyNotFound
}

func newKeys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func sign(t *testing.T, priv ed25519.PrivateKey, kid string, exp time.Time, customerID string) string {
	t.Helper()
	claims := jwtlib.MapClaims{
		"sub":  uuid.NewString(),
		"role": "CLIENT",
		"exp":  exp.Unix(),
		"iss":  "auth-service",
	}
	if customerID != "" {
		claims["customer_id"] = customerID
	}
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// run прогоняет запрос через Auth и возвращает ответ + факт вызова next.
func run(t *testing.T, p port.JWKSProvider, issuer, method, path, authz string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var seen *http.Request
	h := Chain(RequestID(), Auth(p, issuer, discardLog()))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(method, path, nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, seen
}

func code(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var m struct{ Code string }
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	return m.Code
}

func TestMiddlewareAuth_ValidToken_PassesClaims(t *testing.T) {
	pub, priv := newKeys(t)
	cid := uuid.NewString()
	tok := sign(t, priv, "k1", time.Now().Add(time.Minute), cid)

	rec, seen := run(t, fakeJWKS{keys: map[string]ed25519.PublicKey{"k1": pub}}, "auth-service",
		http.MethodGet, "/v1/customers/me", "bearer "+tok)

	if rec.Code != http.StatusNoContent || seen == nil {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body)
	}
	c, ok := reqctx.Claims(seen.Context())
	if !ok || c.CustomerID == nil || c.CustomerID.String() != cid {
		t.Fatalf("claims missing/wrong: %+v", c)
	}
	if reqctx.Token(seen.Context()) != tok {
		t.Fatal("raw token not in ctx")
	}
	if got := seen.Header.Get("Authorization"); got != "Bearer "+tok {
		t.Fatalf("Authorization not normalized: %q", got)
	}
}

func TestMiddlewareAuth_ExpiredToken_Returns401(t *testing.T) {
	pub, priv := newKeys(t)
	tok := sign(t, priv, "k1", time.Now().Add(-time.Minute), "")
	rec, seen := run(t, fakeJWKS{keys: map[string]ed25519.PublicKey{"k1": pub}}, "",
		http.MethodGet, "/v1/customers", "Bearer "+tok)
	if rec.Code != 401 || seen != nil || code(t, rec) != "UNAUTHENTICATED" {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
}

func TestMiddlewareAuth_MissingOrMalformedHeader_Returns401(t *testing.T) {
	for _, h := range []string{"", "Basic abc", "Bearer", "Bearer not.a.jwt"} {
		rec, _ := run(t, fakeJWKS{}, "", http.MethodGet, "/v1/customers", h)
		if rec.Code != 401 {
			t.Errorf("header %q: status = %d", h, rec.Code)
		}
	}
}

func TestMiddlewareAuth_UnknownKid_Returns401(t *testing.T) {
	_, priv := newKeys(t)
	tok := sign(t, priv, "unknown", time.Now().Add(time.Minute), "")
	rec, _ := run(t, fakeJWKS{}, "", http.MethodGet, "/v1/customers", "Bearer "+tok)
	if rec.Code != 401 {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareAuth_WrongSignature_Returns401(t *testing.T) {
	pub, _ := newKeys(t)
	_, otherPriv := newKeys(t)
	tok := sign(t, otherPriv, "k1", time.Now().Add(time.Minute), "")
	rec, _ := run(t, fakeJWKS{keys: map[string]ed25519.PublicKey{"k1": pub}}, "", http.MethodGet, "/v1/customers", "Bearer "+tok)
	if rec.Code != 401 {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareAuth_JWKSUnavailable_Returns503(t *testing.T) {
	_, priv := newKeys(t)
	tok := sign(t, priv, "k1", time.Now().Add(time.Minute), "")
	rec, _ := run(t, fakeJWKS{err: port.ErrJWKSUnavailable}, "", http.MethodGet, "/v1/customers", "Bearer "+tok)
	if rec.Code != 503 || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("status=%d headers=%v", rec.Code, rec.Header())
	}
}

func TestMiddlewareAuth_IssuerMismatch_Returns401(t *testing.T) {
	pub, priv := newKeys(t)
	tok := sign(t, priv, "k1", time.Now().Add(time.Minute), "")
	rec, _ := run(t, fakeJWKS{keys: map[string]ed25519.PublicKey{"k1": pub}}, "someone-else",
		http.MethodGet, "/v1/customers", "Bearer "+tok)
	if rec.Code != 401 {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMiddlewareAuth_PublicRoutes_SkipAuth(t *testing.T) {
	for _, p := range []string{"/v1/auth/register", "/v1/auth/login", "/v1/auth/refresh", "/v1/auth/logout"} {
		rec, seen := run(t, fakeJWKS{}, "", http.MethodPost, p, "Bearer garbage")
		if rec.Code != http.StatusNoContent || seen == nil {
			t.Errorf("%s: status = %d", p, rec.Code)
		}
		if seen != nil && seen.Header.Get("Authorization") != "" {
			t.Errorf("%s: Authorization must be stripped on public routes", p)
		}
	}
}

func TestMiddlewareAuth_OfficersIsProtected(t *testing.T) {
	rec, seen := run(t, fakeJWKS{}, "", http.MethodPost, "/v1/auth/officers", "")
	if rec.Code != 401 || seen != nil {
		t.Fatalf("officers must require JWT, status = %d", rec.Code)
	}
}

func TestMiddlewareAuth_PublicPathWrongMethod_IsProtected(t *testing.T) {
	rec, _ := run(t, fakeJWKS{}, "", http.MethodGet, "/v1/auth/login", "")
	if rec.Code != 401 {
		t.Fatalf("status = %d", rec.Code)
	}
}
