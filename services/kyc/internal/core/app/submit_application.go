package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type SubmitApplicationUseCase struct {
	repo         domain.ApplicationRepository
	customerCli  domain.CustomerClient
	documentCli  domain.DocumentClient
	kafka        domain.EventPublisher
	kafkaTopic   string
	requiredDocs []string
}

func NewSubmitApplicationUseCase(
	repo domain.ApplicationRepository,
	custCli domain.CustomerClient,
	docCli domain.DocumentClient,
	kafka domain.EventPublisher,
	kafkaTopic string,
	requiredDocs []string,
) *SubmitApplicationUseCase {
	return &SubmitApplicationUseCase{
		repo:         repo,
		customerCli:  custCli,
		documentCli:  docCli,
		kafka:        kafka,
		kafkaTopic:   kafkaTopic,
		requiredDocs: requiredDocs,
	}
}

type SubmitApplicationInput struct {
	CustomerID uuid.UUID
}

func (uc *SubmitApplicationUseCase) SubmitApplication(ctx context.Context, in SubmitApplicationInput) (domain.Application, error) {
	// 1. Достаем claims (чтобы взять user_id для аудита)
	claims, err := authjwt.ClaimsFromContext(ctx)
	if err != nil {
		return domain.Application{}, domain.ErrAccessDenied
	}

	// 2. Проверяем, что профиль заполнен, и забираем Snapshot
	fullName, birthDate, status, err := uc.customerCli.GetCustomerProfile(ctx, in.CustomerID)
	if err != nil {
		return domain.Application{}, fmt.Errorf("get customer profile: %w", err)
	}
	if status != "PROFILE_FILLED" {
		return domain.Application{}, domain.ErrProfileNotFilled
	}

	// 3. Проверяем наличие документов
	hasDocs, missing, err := uc.documentCli.HasRequiredDocuments(ctx, in.CustomerID, uc.requiredDocs)
	if err != nil {
		return domain.Application{}, fmt.Errorf("check documents: %w", err)
	}
	if !hasDocs {
		return domain.Application{}, fmt.Errorf("%w: missing %v", domain.ErrDocumentsMissing, missing)
	}

	// 4. Проверяем, нет ли уже активной заявки
	_, err = uc.repo.GetActiveByCustomerID(ctx, in.CustomerID)
	if err == nil {
		return domain.Application{}, domain.ErrActiveApplicationExists
	} else if !errors.Is(err, domain.ErrApplicationNotFound) {
		return domain.Application{}, err
	}

	// 5. Создаем заявку (сразу в SUBMITTED)
	appID, _ := uuid.NewV7()
	app := domain.Application{
		ID:                appID,
		CustomerID:        in.CustomerID,
		Status:            domain.StatusSubmitted,
		FullNameSnapshot:  fullName,
		BirthDateSnapshot: birthDate,
	}

	// В репозитории Create мы сразу же сделаем переход SUBMITTED -> AUTO_CHECKS
	// и запишем события в kyc_events.
	if err := uc.repo.Create(ctx, app, claims.Subject, roles.RoleClient); err != nil {
		return domain.Application{}, err
	}

	// 6. Публикуем событие в Kafka
	_ = uc.kafka.Publish(ctx, uc.kafkaTopic, "kyc.application_submitted", map[string]interface{}{
		"application_id": appID,
		"customer_id":    in.CustomerID,
	})

	// Возвращаем обновленную заявку (можно сходить в БД, но для простоты меняем в памяти)
	app.Status = domain.StatusAutoChecks
	return app, nil
}