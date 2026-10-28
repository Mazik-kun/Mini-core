package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ApplicationRepository interface {
	// Create создает новую заявку (вместе с первым событием kyc_events в одной транзакции).
	Create(ctx context.Context, app Application, actorID uuid.UUID, actorRole string) error
	// GetByID достает заявку по ID.
	GetByID(ctx context.Context, id uuid.UUID) (Application, error)
	// GetActiveByCustomerID ищет незавершенную заявку.
	GetActiveByCustomerID(ctx context.Context, customerID uuid.UUID) (Application, error)
	// Claim берет заявку в работу (оптимистичная блокировка).
	Claim(ctx context.Context, appID, officerID uuid.UUID) error
	// UpdateStatus меняет статус и пишет событие в kyc_events.
	UpdateStatus(ctx context.Context, appID uuid.UUID, from, to Status, actorID uuid.UUID, actorRole, comment string) error
	// List возвращает список заявок (для офицеров).
	List(ctx context.Context, status *Status, limit int, cursor uuid.UUID) ([]Application, error)
}

type CheckRepository interface {
	// SaveCheck сохраняет результат проверки (санкции, возраст).
	SaveCheck(ctx context.Context, appID uuid.UUID, checkType string, result CheckResult) error
	// ListSanctions загружает санкционный список.
	ListSanctions(ctx context.Context) ([]SanctionsEntry, error)
	// ListForAutoChecks достает пачку заявок со статусом AUTO_CHECKS (FOR UPDATE SKIP LOCKED).
	ListForAutoChecks(ctx context.Context, limit int) ([]Application, error)
	// IncrementAttempts увеличивает счетчик попыток. Если превышен лимит — возвращает true.
	IncrementAttempts(ctx context.Context, appID uuid.UUID, maxAttempts int) (exceeded bool, err error)
}

// Внешние gRPC клиенты
type CustomerClient interface {
	// GetCustomerProfile возвращает ФИО и дату рождения (для snapshot).
	GetCustomerProfile(ctx context.Context, customerID uuid.UUID) (fullName string, birthDate time.Time, status string, err error)
}

type DocumentClient interface {
	// HasRequiredDocuments проверяет, загружены ли обязательные документы.
	HasRequiredDocuments(ctx context.Context, customerID uuid.UUID, requiredTypes []string) (hasAll bool, missing []string, err error)
}

// Kafka Producer
type EventPublisher interface {
	Publish(ctx context.Context, topic string, eventType string, payload interface{}) error
}