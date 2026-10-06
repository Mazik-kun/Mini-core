package app

import (
	"context"
	"uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

type GetCustomerStatusUseCase struct {
	repo domain.CustomerRepository
}

func NewGetCustomerStatusUseCase(repo domain.CustomerRepository) *GetCustomerStatusUseCase{
	return &GetCustomerStatusUseCase{repo:repo}
}

type GetCustomerStatusInput struct {
	CustomerID uuid.UUID
	RequesterRole string
	RequesterID uuid.UUID
}

func (uc *GetCustomerStatusUseCase) GetCustomerStatus(ctx context.Context, in GetCustomerStatusInput) (domain.Status, error){
	if in.CustomerID == uuid.Nil() {
		return "", domain.ErrInvalidCustomerID
	}
	if in.RequesterRole == domain.RoleClient && in.CustomerID != in.RequesterID {
		return "", domain.ErrAccessDenied
	}
	customer, err := uc.repo.GetByID(ctx, in.CustomerID)
	if err != nil {
		return "", err
	}
	
	return customer.Status, nil
}