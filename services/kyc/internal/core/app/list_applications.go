package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type ListApplicationsUseCase struct {
	repo domain.ApplicationRepository
}

func NewListApplicationsUseCase(repo domain.ApplicationRepository) *ListApplicationsUseCase {
	return &ListApplicationsUseCase{repo: repo}
}

type ListApplicationsInput struct {
	Status     *domain.Status
	PageSize   int
	PageToken  string
}

func (uc *ListApplicationsUseCase) ListApplications(ctx context.Context, in ListApplicationsInput) ([]domain.Application, error) {
	claims, err := authjwt.ClaimsFromContext(ctx)
	if err != nil {
		return nil, domain.ErrAccessDenied
	}
	if claims.Role != roles.RoleOfficer && claims.Role != roles.RoleAdmin {
		return nil, domain.ErrAccessDenied
	}
	
	// Упрощенная пагинация без токенов, просто по limit (для простоты этапа)
	limit := in.PageSize
	if limit <= 0 {
		limit = 50
	}
	
	return uc.repo.List(ctx, in.Status, limit, uuid.Nil)
}