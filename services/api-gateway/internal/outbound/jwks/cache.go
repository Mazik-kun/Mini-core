package jwks

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type Options struct {
	TTL         time.Duration    // период фоновой рефрешки
	MinInterval time.Duration    // минимальный интервал между on-demand refresh
	Now         func() time.Time // для тестов
	OnRefresh   func(ok bool)    // для метрик
	Log         *slog.Logger
}

// Cache реализует port.JWKSProvider.
type Cache struct {
	fetcher Fetcher
	opts    Options

	mu   sync.RWMutex
	keys map[string]ed25519.PublicKey

	refreshMu   sync.Mutex // сериализует refresh (thundering-herd guard)
	lastRefresh time.Time
	lastErr     error
}

var _ port.JWKSProvider = (*Cache)(nil)

func NewCache(f Fetcher, o Options) *Cache {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Log == nil {
		o.Log = slog.Default()
	}
	return &Cache{fetcher: f, opts: o, keys: map[string]ed25519.PublicKey{}}
}

func (c *Cache) lookup(kid string) (ed25519.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	return k, ok
}

// KeyByID: попадание в кэш → сразу. Промах → один on-demand refresh, но не чаще MinInterval.
func (c *Cache) KeyByID(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	if k, ok := c.lookup(kid); ok {
		return k, nil
	}

	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	// double-check: пока ждали лок, другая горутина могла уже обновить кэш.
	if k, ok := c.lookup(kid); ok {
		return k, nil
	}

	now := c.opts.Now()
	if !c.lastRefresh.IsZero() && now.Sub(c.lastRefresh) < c.opts.MinInterval {
		if c.lastErr != nil {
			return nil, fmt.Errorf("%w: %v", port.ErrJWKSUnavailable, c.lastErr)
		}
		return nil, port.ErrKeyNotFound
	}

	if err := c.refreshLocked(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", port.ErrJWKSUnavailable, err)
	}
	if k, ok := c.lookup(kid); ok {
		return k, nil
	}
	return nil, port.ErrKeyNotFound
}

// Refresh — принудительная загрузка (стартовая и фоновая).
func (c *Cache) Refresh(ctx context.Context) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	return c.refreshLocked(ctx)
}

func (c *Cache) refreshLocked(ctx context.Context) error {
	keys, err := c.fetcher.Fetch(ctx)
	c.lastRefresh = c.opts.Now() // фиксируем и неудачные попытки — иначе шторм запросов к auth
	c.lastErr = err
	if c.opts.OnRefresh != nil {
		c.opts.OnRefresh(err == nil)
	}
	if err != nil {
		return err // старые ключи остаются в кэше
	}
	c.mu.Lock()
	c.keys = keys
	c.mu.Unlock()
	return nil
}

// Run грузит ключи при старте (ошибка не фатальна — gateway поднимется и дозагрузит
// лениво) и обновляет их каждые TTL, пока ctx не отменён.
func (c *Cache) Run(ctx context.Context) {
	if err := c.Refresh(ctx); err != nil {
		c.opts.Log.Warn("initial JWKS load failed; will retry on demand", "error", err)
	}
	t := time.NewTicker(c.opts.TTL)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.Refresh(ctx); err != nil {
				c.opts.Log.Error("background JWKS refresh failed; keeping cached keys", "error", err)
			}
		}
	}
}
