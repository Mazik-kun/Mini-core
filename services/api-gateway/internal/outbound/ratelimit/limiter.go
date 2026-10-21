// Package ratelimit — token bucket поверх Redis (атомарно, через Lua).
package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

//go:embed bucket.lua
var bucketLua string

type Limiter struct {
	rdb      redis.Scripter
	script   *redis.Script
	capacity float64
	rate     float64
	onError  func(op string)
	now      func() time.Time
}

var _ port.RateLimiter = (*Limiter)(nil)

// New: capacity — burst, ratePerSec — скорость пополнения. onError (может быть nil)
// вызывается при любой ошибке Redis (метрика gateway_ratelimit_redis_errors_total).
func New(rdb redis.Scripter, capacity int, ratePerSec float64, onError func(op string)) *Limiter {
	if onError == nil {
		onError = func(string) {}
	}
	return &Limiter{
		rdb:      rdb,
		script:   redis.NewScript(bucketLua),
		capacity: float64(capacity),
		rate:     ratePerSec,
		onError:  onError,
		now:      time.Now,
	}
}

// Allow списывает один токен. Любая ошибка Redis возвращается наверх: решение
// «пропустить» без проверки не принимается никогда (fail-closed).
func (l *Limiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	nowSec := float64(l.now().UnixNano()) / 1e9
	res, err := l.script.Run(ctx, l.rdb, []string{key}, l.capacity, l.rate, nowSec, 1).Slice()
	if err != nil {
		l.onError("eval")
		return false, 0, fmt.Errorf("ratelimit eval: %w", err)
	}
	if len(res) != 2 {
		l.onError("eval")
		return false, 0, fmt.Errorf("ratelimit: unexpected script result %v", res)
	}
	allowed, ok1 := res[0].(int64)
	retryMs, ok2 := res[1].(int64)
	if !ok1 || !ok2 {
		l.onError("eval")
		return false, 0, fmt.Errorf("ratelimit: unexpected result types %T, %T", res[0], res[1])
	}
	return allowed == 1, time.Duration(retryMs) * time.Millisecond, nil
}
