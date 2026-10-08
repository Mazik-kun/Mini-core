package app

import (
	"context"
	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

type GetCustomerUseCase struct {
	repo domain.CustomerRepository
}

func NewGetCustomerUseCase(repo domain.CustomerRepository) *GetCustomerUseCase{
	return &GetCustomerUseCase{repo:repo}
}

type GetCustomerInput struct {
	CustomerID uuid.UUID
	RequesterRole string
	RequesterID uuid.UUID
}

func (uc *GetCustomerUseCase) GetCustomer(ctx context.Context, in GetCustomerInput) (domain.Customer, error){
	if in.CustomerID == uuid.Nil {
		return domain.Customer{}, domain.ErrInvalidCustomerID
	}
	if in.RequesterRole == domain.RoleClient && in.CustomerID != in.RequesterID {
		return domain.Customer{}, domain.ErrAccessDenied
	}
	customer, err := uc.repo.GetByID(ctx, in.CustomerID)
	if err != nil {
		return domain.Customer{}, err
	}
	
	return customer, nil
}