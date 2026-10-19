package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

type GetCustomerStatusUseCase struct {
	repo domain.CustomerRepository
}

func NewGetCustomerStatusUseCase(repo domain.CustomerRepository) *GetCustomerStatusUseCase {
	return &GetCustomerStatusUseCase{repo: repo}
}

type GetCustomerStatusInput struct {
	CustomerID uuid.UUID
}

func (uc *GetCustomerStatusUseCase) GetCustomerStatus(ctx context.Context, in GetCustomerStatusInput) (domain.Status, error) {
	if in.CustomerID == uuid.Nil {
		return "", domain.ErrInvalidCustomerID
	}

	if err := authjwt.RequireCustomerAccess(ctx, in.CustomerID, roles.RoleOfficer, roles.RoleAdmin); err != nil {
		return "", domain.ErrAccessDenied
	}

	customer, err := uc.repo.GetByID(ctx, in.CustomerID)
	if err != nil {
		return "", err
	}
	return customer.Status, nil
}
