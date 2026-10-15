package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

type GetCustomerUseCase struct {
	repo domain.CustomerRepository
}

func NewGetCustomerUseCase(repo domain.CustomerRepository) *GetCustomerUseCase {
	return &GetCustomerUseCase{repo: repo}
}

type GetCustomerInput struct {
	CustomerID uuid.UUID
}

// GetCustomer: CLIENT видит только свой customer_id, OFFICER/ADMIN — любой.
func (uc *GetCustomerUseCase) GetCustomer(ctx context.Context, in GetCustomerInput) (domain.Customer, error) {
	if in.CustomerID == uuid.Nil {
		return domain.Customer{}, domain.ErrInvalidCustomerID
	}

	if err := authjwt.RequireCustomerAccess(ctx, in.CustomerID, roles.RoleOfficer, roles.RoleAdmin); err != nil {
		return domain.Customer{}, domain.ErrAccessDenied
	}

	return uc.repo.GetByID(ctx, in.CustomerID)
}