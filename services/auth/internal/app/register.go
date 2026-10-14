package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

const (
	userRegisteredTopic     = "auth.events"
	userRegisteredEventType = "user_registered"
)

// EventPublisher публикует события в брокер.
// Объявлен здесь, чтобы use case не зависел от конкретной реализации.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, envelope kafka.Envelope) error
}

type RegisterUseCase struct {
	users          domain.UserRepository
	hasher         domain.PasswordHasher
	eventPublisher EventPublisher
}

func NewRegisterUseCase(
	users domain.UserRepository,
	hasher domain.PasswordHasher,
	eventPublisher EventPublisher,
) *RegisterUseCase {
	return &RegisterUseCase{
		users:          users,
		hasher:         hasher,
		eventPublisher: eventPublisher,
	}
}

type RegisterInput struct {
	Email    string
	Password string
}

type RegisterOutput struct {
	UserID uuid.UUID
	Email  string
}

// userRegisteredPayload — payload события user_registered.
type userRegisteredPayload struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
}

func (uc *RegisterUseCase) Register(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if err := validateEmail(email); err != nil {
		return RegisterOutput{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return RegisterOutput{}, err
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := uc.users.Create(ctx, domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         roles.RoleClient,
	})
	if err != nil {
		return RegisterOutput{}, err
	}

	// Упрощение: падение публикации не откатывает регистрацию —
	// пользователь уже сохранён, событие просто потеряется, пишем в лог.
	if err := uc.publishUserRegistered(ctx, user); err != nil {
		slog.Error("не удалось опубликовать user_registered",
			"user_id", user.ID, "error", err)
	}

	return RegisterOutput{UserID: user.ID, Email: user.Email}, nil
}

func (uc *RegisterUseCase) publishUserRegistered(ctx context.Context, user domain.User) error {
	envelope, err := kafka.NewEnvelope(userRegisteredEventType, userRegisteredPayload{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		return fmt.Errorf("собрать конверт: %w", err)
	}
	return uc.eventPublisher.Publish(ctx, userRegisteredTopic, envelope)
}