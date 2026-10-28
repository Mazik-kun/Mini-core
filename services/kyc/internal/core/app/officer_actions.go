package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type OfficerActionsUseCase struct {
	repo       domain.ApplicationRepository
	kafka      domain.EventPublisher
	kafkaTopic string
}

func NewOfficerActionsUseCase(repo domain.ApplicationRepository, kafka domain.EventPublisher, topic string) *OfficerActionsUseCase {
	return &OfficerActionsUseCase{repo: repo, kafka: kafka, kafkaTopic: topic}
}

// ClaimApplication — офицер берет заявку в работу
func (uc *OfficerActionsUseCase) ClaimApplication(ctx context.Context, appID uuid.UUID) (domain.Application, error) {
	claims, err := authjwt.ClaimsFromContext(ctx)
	if err != nil {
		return domain.Application{}, domain.ErrAccessDenied
	}

	app, err := uc.repo.GetByID(ctx, appID)
	if err != nil {
		return domain.Application{}, err
	}
	if app.Status != domain.StatusInReview {
		return domain.Application{}, fmt.Errorf("%w: must be IN_REVIEW", domain.ErrInvalidStatusTransition)
	}

	// Оптимистичная блокировка в БД
	if err := uc.repo.Claim(ctx, appID, claims.Subject); err != nil {
		return domain.Application{}, err
	}

	app.OfficerID = &claims.Subject
	return app, nil
}

// ApproveApplication — одобрение
func (uc *OfficerActionsUseCase) ApproveApplication(ctx context.Context, appID uuid.UUID) (domain.Application, error) {
	app, claims, err := uc.validateOfficerAction(ctx, appID)
	if err != nil {
		return domain.Application{}, err
	}

	if err := uc.repo.UpdateStatus(ctx, appID, app.Status, domain.StatusApproved, claims.Subject, roles.RoleOfficer, ""); err != nil {
		return domain.Application{}, err
	}

	_ = uc.kafka.Publish(ctx, uc.kafkaTopic, "kyc.application_approved", map[string]interface{}{
		"application_id": appID,
		"customer_id":    app.CustomerID,
	})

	app.Status = domain.StatusApproved
	return app, nil
}

// RejectApplication — отклонение
func (uc *OfficerActionsUseCase) RejectApplication(ctx context.Context, appID uuid.UUID, reason string) (domain.Application, error) {
	app, claims, err := uc.validateOfficerAction(ctx, appID)
	if err != nil {
		return domain.Application{}, err
	}

	if err := uc.repo.UpdateStatus(ctx, appID, app.Status, domain.StatusRejected, claims.Subject, roles.RoleOfficer, reason); err != nil {
		return domain.Application{}, err
	}

	_ = uc.kafka.Publish(ctx, uc.kafkaTopic, "kyc.application_rejected", map[string]interface{}{
		"application_id": appID,
		"customer_id":    app.CustomerID,
		"reason":         reason,
	})

	app.Status = domain.StatusRejected
	app.Reason = reason
	return app, nil
}

// RequestMoreDocuments — запросить доп. документы
func (uc *OfficerActionsUseCase) RequestMoreDocuments(ctx context.Context, appID uuid.UUID, message string) (domain.Application, error) {
	app, claims, err := uc.validateOfficerAction(ctx, appID)
	if err != nil {
		return domain.Application{}, err
	}

	if err := uc.repo.UpdateStatus(ctx, appID, app.Status, domain.StatusDocsRequested, claims.Subject, roles.RoleOfficer, message); err != nil {
		return domain.Application{}, err
	}

	_ = uc.kafka.Publish(ctx, uc.kafkaTopic, "kyc.docs_requested", map[string]interface{}{
		"application_id": appID,
		"customer_id":    app.CustomerID,
		"message":        message,
	})

	app.Status = domain.StatusDocsRequested
	return app, nil
}

// Вспомогательная функция для проверки прав
func (uc *OfficerActionsUseCase) validateOfficerAction(ctx context.Context, appID uuid.UUID) (domain.Application, authjwt.Claims, error) {
	claims, err := authjwt.ClaimsFromContext(ctx)
	if err != nil || claims.Role != roles.RoleOfficer {
		return domain.Application{}, claims, domain.ErrAccessDenied
	}

	app, err := uc.repo.GetByID(ctx, appID)
	if err != nil {
		return domain.Application{}, claims, err
	}

	// Заявка должна быть в работе и принадлежать этому офицеру
	if app.Status != domain.StatusInReview {
		return domain.Application{}, claims, fmt.Errorf("%w: must be IN_REVIEW", domain.ErrInvalidStatusTransition)
	}
	if app.OfficerID == nil || *app.OfficerID != claims.Subject {
		return domain.Application{}, claims, domain.ErrAccessDenied
	}

	return app, claims, nil
}