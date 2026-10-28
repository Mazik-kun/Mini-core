package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type GetApplicationUseCase struct {
	repo domain.ApplicationRepository
}

func NewGetApplicationUseCase(repo domain.ApplicationRepository) *GetApplicationUseCase {
	return &GetApplicationUseCase{repo: repo}
}

func (uc *GetApplicationUseCase) GetApplication(ctx context.Context, appID uuid.UUID) (domain.Application, error) {
	app, err := uc.repo.GetByID(ctx, appID)
	if err != nil {
		return domain.Application{}, err
	}
	claims, err := authjwt.ClaimsFromContext(ctx)
	if err != nil {
		return domain.Application{}, domain.ErrAccessDenied
	}
	// CLIENT видит только свои заявки. OFFICER и ADMIN видят все.
	if claims.Role == roles.RoleClient {
		if claims.CustomerID == nil || *claims.CustomerID != app.CustomerID {
			return domain.Application{}, domain.ErrAccessDenied
		}
	}
	return app, nil
}