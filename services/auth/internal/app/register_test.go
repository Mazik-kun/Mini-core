package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/auth/internal/app"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

type fakeRepo struct {
	users map[string]domain.User
}

func (r *fakeRepo) Create(_ context.Context, u domain.User) (domain.User, error) {
	if _, ok := r.users[u.Email]; ok {
		return domain.User{}, domain.ErrEmailAlreadyExists
	}
	u.ID = uuid.New()
	r.users[u.Email] = u
	return u, nil
}

func (r *fakeRepo) GetByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := r.users[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (r *fakeRepo) ExistsByRole(_ context.Context, role string) (bool, error) {
	for _, u := range r.users {
		if u.Role == role {
			return true, nil
		}
	}
	return false, nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error)       { return "hashed:" + plain, nil }
func (fakeHasher) Verify(plain, hash string) (bool, error) { return hash == "hashed:"+plain, nil }

type publishedEvent struct {
	topic    string
	envelope kafka.Envelope
}

type fakePublisher struct {
	events []publishedEvent
	err    error
}

func (p *fakePublisher) Publish(_ context.Context, topic string, envelope kafka.Envelope) error {
	p.events = append(p.events, publishedEvent{topic: topic, envelope: envelope})
	return p.err
}

func newUseCase() (*app.RegisterUseCase, *fakeRepo, *fakePublisher) {
	repo := &fakeRepo{users: map[string]domain.User{}}
	publisher := &fakePublisher{}
	return app.NewRegisterUseCase(repo, fakeHasher{}, publisher), repo, publisher
}

func TestRegister_Success(t *testing.T) {
	uc, _, _ := newUseCase()
	out, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "Test@Example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Email != "test@example.com" {
		t.Errorf("email should be lowercased, got %s", out.Email)
	}
	if out.UserID == uuid.Nil {
		t.Error("user id must not be nil")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	uc, _, _ := newUseCase()
	in := app.RegisterInput{Email: "a@b.com", Password: "password123"}
	if _, err := uc.Register(context.Background(), in); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := uc.Register(context.Background(), in)
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("want ErrEmailAlreadyExists, got %v", err)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	uc, _, _ := newUseCase()
	_, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "not-an-email",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Errorf("want ErrInvalidEmail, got %v", err)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	uc, _, _ := newUseCase()
	_, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "a@b.com",
		Password: "short",
	})
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Errorf("want ErrWeakPassword, got %v", err)
	}
}

func TestRegister_PublishesUserRegisteredEvent(t *testing.T) {
	uc, _, publisher := newUseCase()

	if _, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "a@b.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("опубликовано %d событий, ожидалось 1", len(publisher.events))
	}

	event := publisher.events[0]
	if event.topic != "auth.events" {
		t.Errorf("topic = %q, ожидалось %q", event.topic, "auth.events")
	}
	if event.envelope.EventType != "user_registered" {
		t.Errorf("event_type = %q, ожидалось %q", event.envelope.EventType, "user_registered")
	}

	var payload struct {
		UserID uuid.UUID `json:"user_id"`
		Email  string    `json:"email"`
		Role   string    `json:"role"`
	}
	if err := event.envelope.UnmarshalPayload(&payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Email != "a@b.com" {
		t.Errorf("payload.Email = %q, ожидалось %q", payload.Email, "a@b.com")
	}
	if payload.Role != "CLIENT" {
		t.Errorf("payload.Role = %q, ожидалось %q", payload.Role, "CLIENT")
	}
}

func TestRegister_PublishFailureDoesNotFailRegistration(t *testing.T) {
	uc, repo, publisher := newUseCase()
	publisher.err = errors.New("kafka down")

	if _, err := uc.Register(context.Background(), app.RegisterInput{
		Email:    "a@b.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("регистрация должна пройти даже при ошибке публикации: %v", err)
	}

	if _, ok := repo.users["a@b.com"]; !ok {
		t.Error("пользователь должен быть сохранён")
	}
}