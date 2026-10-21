package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

type fakeLimiter struct {
	allowed bool
	retry   time.Duration
	err     error
	keys    []string
}

func (f *fakeLimiter) Allow(_ context.Context, key string) (bool, time.Duration, error) {
	f.keys = append(f.keys, key)
	return f.allowed, f.retry, f.err
}

func rlHandler(user, ip *fakeLimiter) (http.Handler, *bool) {
	called := false
	m := observability.NewMetrics(prometheus.NewRegistry())
	h := RateLimit(user, ip, false, m, discardLog())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	return h, &called
}

func TestMiddlewareRateLimit_Allowed_CallsNext(t *testing.T) {
	user, ip := &fakeLimiter{allowed: true}, &fakeLimiter{allowed: true}
	h, called := rlHandler(user, ip)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.7:5555"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !*called || rec.Code != 204 {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(ip.keys) != 1 || ip.keys[0] != "rl:ip:203.0.113.7" || len(user.keys) != 0 {
		t.Fatalf("wrong limiter/key: ip=%v user=%v", ip.keys, user.keys)
	}
}

func TestMiddlewareRateLimit_AuthenticatedUsesUserKey(t *testing.T) {
	user, ip := &fakeLimiter{allowed: true}, &fakeLimiter{allowed: true}
	h, _ := rlHandler(user, ip)
	sub := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/customers", nil)
	req = req.WithContext(reqctx.WithClaims(req.Context(), authjwt.Claims{Subject: sub}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if len(user.keys) != 1 || user.keys[0] != "rl:user:"+sub.String() || len(ip.keys) != 0 {
		t.Fatalf("wrong limiter/key: ip=%v user=%v", ip.keys, user.keys)
	}
}

func TestMiddlewareRateLimit_Denied_Returns429WithRetryAfter(t *testing.T) {
	h, called := rlHandler(&fakeLimiter{}, &fakeLimiter{allowed: false, retry: 1500 * time.Millisecond})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil))

	if *called || rec.Code != 429 {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "2" {
		t.Fatalf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	var env struct {
		Code    string
		Details []map[string]any
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Code != "RESOURCE_EXHAUSTED" || len(env.Details) != 1 ||
		env.Details[0]["@type"] != "type.googleapis.com/google.rpc.RetryInfo" {
		t.Fatalf("unexpected body: %s", rec.Body)
	}
}

func TestMiddlewareRateLimit_RedisError_FailsClosedWith503(t *testing.T) {
	h, called := rlHandler(&fakeLimiter{}, &fakeLimiter{err: errors.New("redis: connection refused")})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil))

	if *called {
		t.Fatal("request must NOT pass when limiter errors (fail-closed)")
	}
	if rec.Code != 503 || rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("status=%d retry-after=%q", rec.Code, rec.Header().Get("Retry-After"))
	}
	var env struct{ Code, Message string }
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Code != "UNAVAILABLE" || env.Message != "rate limiter unavailable" {
		t.Fatalf("unexpected body: %s", rec.Body)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.1")
	if got := ClientIP(r, false); got != "10.0.0.1" {
		t.Errorf("untrusted XFF must be ignored, got %s", got)
	}
	if got := ClientIP(r, true); got != "198.51.100.9" {
		t.Errorf("trusted XFF, got %s", got)
	}
}

func TestRouteLabel_LowCardinality(t *testing.T) {
	cases := map[string]string{
		"/v1/auth/login":                  "/v1/auth/login",
		"/v1/customers":                   "/v1/customers",
		"/v1/customers/me":                "/v1/customers/me",
		"/v1/customers/me/status":         "/v1/customers/me/status",
		"/v1/customers/0198-abc":          "/v1/customers/{customer_id}",
		"/v1/customers/0198-abc/whatever": "unmatched",
		"/random/scanner/path":            "unmatched",
	}
	for path, want := range cases {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if got := RouteLabel(r); got != want {
			t.Errorf("RouteLabel(%s) = %s, want %s", path, got, want)
		}
	}
}
