package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newLimiter(t *testing.T, capacity int, rate float64) (*Limiter, *miniredis.Miniredis, *time.Time, *int) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = rdb.Close() })

	errCount := 0
	l := New(rdb, capacity, rate, func(string) { errCount++ })
	now := time.Unix(1_800_000_000, 0)
	l.now = func() time.Time { return now }
	return l, mr, &now, &errCount
}

func TestLimiter_BurstThenDeny(t *testing.T) {
	l, _, _, _ := newLimiter(t, 2, 1)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		ok, _, err := l.Allow(ctx, "rl:test")
		if err != nil || !ok {
			t.Fatalf("request %d: ok=%v err=%v", i, ok, err)
		}
	}
	ok, retry, err := l.Allow(ctx, "rl:test")
	if err != nil || ok {
		t.Fatalf("3rd request must be denied: ok=%v err=%v", ok, err)
	}
	if retry <= 0 || retry > time.Second {
		t.Fatalf("retryAfter = %v, want (0, 1s]", retry)
	}
}

func TestLimiter_RefillsOverTime(t *testing.T) {
	l, _, now, _ := newLimiter(t, 2, 1)
	ctx := context.Background()
	_, _, _ = l.Allow(ctx, "k")
	_, _, _ = l.Allow(ctx, "k")
	if ok, _, _ := l.Allow(ctx, "k"); ok {
		t.Fatal("bucket should be empty")
	}

	*now = now.Add(1100 * time.Millisecond)
	if ok, _, err := l.Allow(ctx, "k"); err != nil || !ok {
		t.Fatalf("after refill: ok=%v err=%v", ok, err)
	}
}

func TestLimiter_KeysAreIndependent(t *testing.T) {
	l, _, _, _ := newLimiter(t, 1, 1)
	ctx := context.Background()
	if ok, _, _ := l.Allow(ctx, "rl:user:a"); !ok {
		t.Fatal("a first must pass")
	}
	if ok, _, _ := l.Allow(ctx, "rl:user:b"); !ok {
		t.Fatal("b must have its own bucket")
	}
}

func TestLimiter_RedisDown_ReturnsErrorAndCountsMetric(t *testing.T) {
	l, mr, _, errCount := newLimiter(t, 1, 1)
	mr.Close()

	ok, _, err := l.Allow(context.Background(), "k")
	if err == nil || ok {
		t.Fatalf("must fail closed: ok=%v err=%v", ok, err)
	}
	if *errCount != 1 {
		t.Fatalf("redis error metric callback count = %d", *errCount)
	}
}
