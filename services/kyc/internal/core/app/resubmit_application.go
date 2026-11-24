package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type ResubmitApplicationUseCase struct {
	repo         domain.ApplicationRepository
	documentCli  domain.DocumentClient
	kafka        domain.EventPublisher
	kafkaTopic   string
	requiredDocs []string
}

func NewResubmitApplicationUseCase(
	repo domain.ApplicationRepository,
	docCli domain.DocumentClient,
	kafka domain.EventPublisher,
	kafkaTopic string,
	requiredDocs []string,
) *ResubmitApplicationUseCase {
	return &ResubmitApplicationUseCase{
		repo:         repo,
		documentCli:  docCli,
		kafka:        kafka,
		kafkaTopic:   kafkaTopic,
		requiredDocs: requiredDocs,
	}
}

func (uc *ResubmitApplicationUseCase) ResubmitApplication(ctx context.Context, appID uuid.UUID) (domain.Application, error) {
	// 1. Достаем claims: повторно подать заявку может только клиент
	claims, err := authjwt.ClaimsFromContext(ctx)
	if err != nil {
		return domain.Application{}, domain.ErrAccessDenied
	}
	if claims.Role != roles.RoleClient {
		return domain.Application{}, domain.ErrAccessDenied
	}

	// 2. Достаем заявку (ErrApplicationNotFound приходит из репозитория как есть
	app, err := uc.repo.GetByID(ctx, appID)
	if err != nil {
		return domain.Application{}, err
	}

	// 3. Заявка должна принадлежать этому клиенту (защита от IDOR)
	if claims.CustomerID == nil || app.CustomerID != *claims.CustomerID {
		return domain.Application{}, domain.ErrAccessDenied
	}

	// 4. Проверяем всю цепочку переходов ДО записи в БД:
	//    DOCS_REQUESTED -> SUBMITTED -> AUTO_CHECKS
	if !app.Status.CanTransitionTo(domain.StatusSubmitted) ||
		!domain.StatusSubmitted.CanTransitionTo(domain.StatusAutoChecks) {
		return domain.Application{}, fmt.Errorf("%w: %s -> %s", domain.ErrInvalidStatusTransition, app.Status, domain.StatusSubmitted)
	}

	// 5. Проверяем, что клиент догрузил обязательные документы (как в submit)
	hasDocs, missing, err := uc.documentCli.HasRequiredDocuments(ctx, app.CustomerID, uc.requiredDocs)
	if err != nil {
		return domain.Application{}, fmt.Errorf("check documents: %w", err)
	}
	if !hasDocs {
		return domain.Application{}, fmt.Errorf("%w: missing %v", domain.ErrDocumentsMissing, missing)
	}

	// 6. DOCS_REQUESTED -> SUBMITTED
	if err := uc.repo.UpdateStatus(ctx, app.ID, app.Status, domain.StatusSubmitted, claims.Subject, roles.RoleClient, "client resubmitted documents"); err != nil {
		return domain.Application{}, err
	}

	// 7. SUBMITTED -> AUTO_CHECKS: при первой подаче этот переход делает repo.Create,
	// здесь его нужно сделать явно, иначе воркер автопроверок заявку не подхватит.
	if err := uc.repo.UpdateStatus(ctx, app.ID, domain.StatusSubmitted, domain.StatusAutoChecks, claims.Subject, roles.RoleClient, "auto checks started"); err != nil {
		return domain.Application{}, err
	}

	// 8. Событие в Kafka (как в submit, ошибка публикации не валит операцию)
	_ = uc.kafka.Publish(ctx, uc.kafkaTopic, "kyc.application_submitted", map[string]interface{}{
		"application_id": app.ID,
		"customer_id":    app.CustomerID,
	})

	app.Status = domain.StatusAutoChecks
	return app, nil
}