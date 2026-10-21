package jwks

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type fakeFetcher struct {
	mu    sync.Mutex
	calls atomic.Int32
	keys  map[string]ed25519.PublicKey
	err   error
}

func (f *fakeFetcher) Fetch(context.Context) (map[string]ed25519.PublicKey, error) {
	f.calls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]ed25519.PublicKey, len(f.keys))
	for k, v := range f.keys {
		out[k] = v
	}
	return out, nil
}

func (f *fakeFetcher) set(keys map[string]ed25519.PublicKey, err error) {
	f.mu.Lock()
	f.keys, f.err = keys, err
	f.mu.Unlock()
}

func pubKey(t *testing.T) ed25519.PublicKey {
	t.Helper()
	p, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newCache(f Fetcher, clk *clock) *Cache {
	return NewCache(f, Options{TTL: time.Hour, MinInterval: time.Minute, Now: clk.now})
}

func TestCache_UnknownKid_OnDemandRefreshFindsKey(t *testing.T) {
	k := pubKey(t)
	f := &fakeFetcher{keys: map[string]ed25519.PublicKey{"k1": k}}
	c := newCache(f, &clock{t: time.Unix(1000, 0)})

	got, err := c.KeyByID(context.Background(), "k1")
	if err != nil || !got.Equal(k) {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, err := c.KeyByID(context.Background(), "k1"); err != nil || f.calls.Load() != 1 {
		t.Fatalf("second lookup must hit cache, fetches=%d err=%v", f.calls.Load(), err)
	}
}

func TestCache_MinIntervalGuard(t *testing.T) {
	f := &fakeFetcher{keys: map[string]ed25519.PublicKey{"k1": pubKey(t)}}
	clk := &clock{t: time.Unix(1000, 0)}
	c := newCache(f, clk)

	if _, err := c.KeyByID(context.Background(), "nope"); !errors.Is(err, port.ErrKeyNotFound) {
		t.Fatalf("err = %v", err)
	}
	// Атакующий шлёт токены со случайными kid: auth не должен получать шторм запросов.
	for i := 0; i < 10; i++ {
		_, _ = c.KeyByID(context.Background(), fmt.Sprintf("rnd-%d", i))
	}
	if f.calls.Load() != 1 {
		t.Fatalf("fetches = %d, want 1 within min interval", f.calls.Load())
	}

	clk.t = clk.t.Add(2 * time.Minute) // guard истёк, и в JWKS появился новый ключ (ротация)
	newKey := pubKey(t)
	f.set(map[string]ed25519.PublicKey{"k2": newKey}, nil)
	got, err := c.KeyByID(context.Background(), "k2")
	if err != nil || !got.Equal(newKey) || f.calls.Load() != 2 {
		t.Fatalf("after interval: key=%v err=%v fetches=%d", got, err, f.calls.Load())
	}
}

func TestCache_FetchError_ReturnsUnavailable_KeepsOldKeys(t *testing.T) {
	k := pubKey(t)
	f := &fakeFetcher{keys: map[string]ed25519.PublicKey{"k1": k}}
	clk := &clock{t: time.Unix(1000, 0)}
	c := newCache(f, clk)
	if err := c.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}

	f.set(nil, errors.New("connection refused"))
	clk.t = clk.t.Add(time.Hour)
	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("expected refresh error")
	}
	if got, err := c.KeyByID(context.Background(), "k1"); err != nil || !got.Equal(k) {
		t.Fatalf("old key must survive failed refresh: %v %v", got, err)
	}
	if _, err := c.KeyByID(context.Background(), "unknown"); !errors.Is(err, port.ErrJWKSUnavailable) {
		t.Fatalf("err = %v, want ErrJWKSUnavailable", err)
	}
}

func TestCache_ThunderingHerd_SingleFetch(t *testing.T) {
	f := &fakeFetcher{keys: map[string]ed25519.PublicKey{"k1": pubKey(t)}}
	c := newCache(f, &clock{t: time.Unix(1000, 0)})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.KeyByID(context.Background(), "k1"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if f.calls.Load() != 1 {
		t.Fatalf("fetches = %d, want 1", f.calls.Load())
	}
}

func TestHTTPFetcher_ParsesEd25519Keys(t *testing.T) {
	k := pubKey(t)
	body := fmt.Sprintf(`{"keys":[
		{"kty":"RSA","kid":"rsa","n":"x","e":"AQAB"},
		{"kty":"OKP","crv":"Ed25519","kid":"auth-1","use":"sig","alg":"EdDSA","x":%q}
	]}`, base64.RawURLEncoding.EncodeToString(k))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	keys, err := NewHTTPFetcher(srv.URL, time.Second).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || !keys["auth-1"].Equal(k) {
		t.Fatalf("unexpected keys: %v", keys)
	}
}

func TestHTTPFetcher_Non200_IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := NewHTTPFetcher(srv.URL, time.Second).Fetch(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
