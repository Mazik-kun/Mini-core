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

// DedupeStore отмечает события как обработанные и сообщает о повторах.
type DedupeStore interface {
	// MarkProcessed возвращает alreadyProcessed = true, если событие уже обрабатывалось.
	MarkProcessed(ctx context.Context, eventID string) (alreadyProcessed bool, err error)
}

// RedisDedupeStore — реализация DedupeStore на Redis.
type RedisDedupeStore struct {
	client *redis.Client
}

// NewRedisDedupeStore создаёт хранилище дедупликации.
func NewRedisDedupeStore(client *redis.Client) *RedisDedupeStore {
	return &RedisDedupeStore{client: client}
}

// MarkProcessed ставит ключ processed_event:<event_id> через SET NX с TTL 7 дней.
// alreadyProcessed = true, если ключ уже существовал.
func (store *RedisDedupeStore) MarkProcessed(ctx context.Context, eventID string) (bool, error) {
	key := processedEventKeyPrefix + eventID

	wasSet, err := store.client.SetNX(ctx, key, "1", processedEventTTL).Result()
	if err != nil {
		return false, err
	}
	return !wasSet, nil
}