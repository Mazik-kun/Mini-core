package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

const testCustomerEventsTopic = "customer.events"

type publishCall struct {
	Topic    string
	Envelope kafkautil.Envelope
}

type fakePublisher struct {
	calls []publishCall
	err   error
}

func (p *fakePublisher) Publish(ctx context.Context, topic string, envelope kafkautil.Envelope) error {
	p.calls = append(p.calls, publishCall{Topic: topic, Envelope: envelope})
	return p.err
}

type countingRepo struct {
	createCalls int
	lastUserID  uuid.UUID
	createErr   error
}

func (r *countingRepo) Create(ctx context.Context, c domain.Customer) (domain.Customer, error) {
	r.createCalls++
	r.lastUserID = c.UserID
	if r.createErr != nil {
		return domain.Customer{}, r.createErr
	}
	c.ID = uuid.Must(uuid.NewV7())
	return c, nil
}

func (r *countingRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (r *countingRepo) GetByUserID(ctx context.Context, id uuid.UUID) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (r *countingRepo) Update(ctx context.Context, c domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (r *countingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, s domain.Status) error {
	return nil
}

func (r *countingRepo) List(ctx context.Context, f domain.ListFilter) ([]domain.Customer, error) {
	return nil, nil
}

func TestCreateCustomer_PublishesCustomerCreated(t *testing.T) {
	repo := &countingRepo{}
	pub := &fakePublisher{}
	uc := NewCreateCustomerUseCase(repo, pub, testCustomerEventsTopic)

	userID := uuid.Must(uuid.NewV7())
	created, err := uc.CreateCustomer(context.Background(), CreateCustomerInput{UserID: userID})
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}

	if repo.createCalls != 1 {
		t.Fatalf("Create должен быть вызван 1 раз, вызван %d", repo.createCalls)
	}
	if len(pub.calls) != 1 {
		t.Fatalf("ожидался 1 вызов Publish, получили %d", len(pub.calls))
	}
	if pub.calls[0].Topic != testCustomerEventsTopic {
		t.Errorf("topic: ожидался %q, получили %q", testCustomerEventsTopic, pub.calls[0].Topic)
	}
	if pub.calls[0].Envelope.EventType != customerCreatedEventType {
		t.Errorf("event_type: ожидался %q, получили %q",
			customerCreatedEventType, pub.calls[0].Envelope.EventType)
	}
	if created.ID == uuid.Nil {
		t.Errorf("customer_id пустой")
	}
}

func TestCreateCustomer_PublishError_DoesNotFail(t *testing.T) {
	repo := &countingRepo{}
	pub := &fakePublisher{err: errors.New("kafka down")}
	uc := NewCreateCustomerUseCase(repo, pub, testCustomerEventsTopic)

	userID := uuid.Must(uuid.NewV7())
	_, err := uc.CreateCustomer(context.Background(), CreateCustomerInput{UserID: userID})
	if err != nil {
		t.Fatalf("падение publish не должно валить создание профиля: %v", err)
	}
	if repo.createCalls != 1 {
		t.Errorf("профиль всё равно должен быть создан")
	}
}

func TestCreateCustomer_EmptyUserID(t *testing.T) {
	repo := &countingRepo{}
	pub := &fakePublisher{}
	uc := NewCreateCustomerUseCase(repo, pub, testCustomerEventsTopic)

	_, err := uc.CreateCustomer(context.Background(), CreateCustomerInput{})
	if !errors.Is(err, domain.ErrInvalidCustomerID) {
		t.Fatalf("ожидалась ErrInvalidCustomerID, получили %v", err)
	}
	if len(pub.calls) != 0 {
		t.Errorf("не должно быть publish при пустом user_id")
	}
}
