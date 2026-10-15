package kafka

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	processedEventKeyPrefix = "processed_event:"
	processedEventTTL       = 7 * 24 * time.Hour
)

type DedupeStore interface {
	// MarkProcessed возвращает alreadyProcessed = true, если событие уже обрабатывалось.
	MarkProcessed(ctx context.Context, eventID string) (alreadyProcessed bool, err error)
}

type RedisDedupeStore struct {
	client *redis.Client
}

func NewRedisDedupeStore(client *redis.Client) *RedisDedupeStore {
	return &RedisDedupeStore{client: client}
}

// NewRedisDedupeStoreFromAddr открывает соединение с Redis самостоятельно.
func NewRedisDedupeStoreFromAddr(addr string) *RedisDedupeStore {
	client := redis.NewClient(&redis.Options{Addr: addr})
	return &RedisDedupeStore{client: client}
}

// Close закрывает соединение с Redis.
func (store *RedisDedupeStore) Close() error {
	return store.client.Close()
}

func (store *RedisDedupeStore) MarkProcessed(ctx context.Context, eventID string) (bool, error) {
	key := processedEventKeyPrefix + eventID
	wasSet, err := store.client.SetNX(ctx, key, "1", processedEventTTL).Result()
	if err != nil {
		return false, err
	}
	return !wasSet, nil
}