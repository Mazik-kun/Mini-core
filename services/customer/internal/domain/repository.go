package domain

import (
	"context"
	"uuid"

)

type CustomerRepository interface {
	Create(ctx context.Context, customer Customer)(Customer, error)
	GetByID(ctx context.Context, id uuid.UUID) (Customer, error)
	GetByUserID(ctx context.Context, id uuid.UUID)(Customer, error)
	Update(ctx context.Context, customer Customer)(Customer, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	List(ctx context.Context, list ListFilter)([]Customer, error)
}
type ListFilter struct {
	Statuses []Status
	Limit int
	Offset int
}