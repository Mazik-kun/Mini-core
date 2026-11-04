package app

import (
	"context"
	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

type GetAccountUseCase struct { repo domain.AccountRepository }

func NewGetAccountUseCase(repo domain.AccountRepository) *GetAccountUseCase { return &GetAccountUseCase{repo: repo} }

func (uc *GetAccountUseCase) GetByID(ctx context.Context, id uuid.UUID) (domain.Account, error) {
	return uc.repo.GetByID(ctx, id)
}