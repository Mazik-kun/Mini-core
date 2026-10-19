package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"

	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

type fakeDedupeStore struct {
	mu        sync.Mutex
	processed map[string]bool
}

func newFakeDedupeStore() *fakeDedupeStore {
	return &fakeDedupeStore{processed: map[string]bool{}}
}

func (s *fakeDedupeStore) MarkProcessed(_ context.Context, eventID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processed[eventID] {
		return true, nil
	}
	s.processed[eventID] = true
	return false, nil
}

type countingRepo struct {
	mu        sync.Mutex
	createCnt int
}

func (r *countingRepo) Create(_ context.Context, c domain.Customer) (domain.Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCnt++
	c.ID = uuid.New()
	return c, nil
}

func (r *countingRepo) GetByID(_ context.Context, _ uuid.UUID) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *countingRepo) GetByUserID(_ context.Context, _ uuid.UUID) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *countingRepo) Update(_ context.Context, c domain.Customer) (domain.Customer, error) {
	return c, nil
}

func (r *countingRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.Status) error {
	return nil
}

func (r *countingRepo) List(_ context.Context, _ domain.ListFilter) ([]domain.Customer, error) {
	return nil, nil
}

type fakePublisher struct {
	calls []kafkautil.Envelope
	err   error
}

func (p *fakePublisher) Publish(ctx context.Context, topic string, envelope kafkautil.Envelope) error {
	p.calls = append(p.calls, envelope)
	return p.err
}

// Повторный event с тем же event_id → CreateCustomer вызван один раз.
func TestConsumer_DedupeSkipsRepeatedEvent(t *testing.T) {
	repo := &countingRepo{}
	publisher := &fakePublisher{}
	createCustomer := app.NewCreateCustomerUseCase(repo, publisher, "customer.events")
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	consumer := NewConsumer(nil, "test", "auth.events", newFakeDedupeStore(), createCustomer, log)

	userID := uuid.New()
	payloadBytes, _ := json.Marshal(UserRegisteredPayload{
		UserID: userID,
		Role:   "CLIENT",
	})
	envelope := kafkautil.Envelope{
		EventID:   uuid.New().String(),
		EventType: UserRegisteredEventType,
		Payload:   payloadBytes,
	}

	if err := consumer.handleEnvelope(context.Background(), envelope); err != nil {
		t.Fatalf("первая обработка: %v", err)
	}
	if err := consumer.handleEnvelope(context.Background(), envelope); err != nil {
		t.Fatalf("вторая обработка: %v", err)
	}

	if repo.createCnt != 1 {
		t.Fatalf("CreateCustomer вызван %d раз, ожидали 1", repo.createCnt)
	}
}

// Событие с ролью OFFICER не создаёт клиента.
func TestConsumer_SkipsNonClientRole(t *testing.T) {
	repo := &countingRepo{}
	publisher := &fakePublisher{}
	createCustomer := app.NewCreateCustomerUseCase(repo, publisher, "customer.events")
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	consumer := NewConsumer(nil, "test", "auth.events", newFakeDedupeStore(), createCustomer, log)

	payloadBytes, _ := json.Marshal(UserRegisteredPayload{
		UserID: uuid.New(),
		Role:   "OFFICER",
	})
	envelope := kafkautil.Envelope{
		EventID:   uuid.New().String(),
		EventType: UserRegisteredEventType,
		Payload:   payloadBytes,
	}

	if err := consumer.handleEnvelope(context.Background(), envelope); err != nil {
		t.Fatalf("handleEnvelope: %v", err)
	}
	if repo.createCnt != 0 {
		t.Fatalf("CreateCustomer вызван %d раз, ожидали 0", repo.createCnt)
	}
}
