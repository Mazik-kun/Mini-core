package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

type HasRequiredDocumentsUseCase struct {
	repo            domain.DocumentRepository
	defaultRequired []domain.Type
}

func NewHasRequiredDocumentsUseCase(repo domain.DocumentRepository, defaultRequired []domain.Type) *HasRequiredDocumentsUseCase {
	return &HasRequiredDocumentsUseCase{repo: repo, defaultRequired: defaultRequired}
}

type HasRequiredDocumentsInput struct {
	CustomerID uuid.UUID
	Required   []domain.Type // пусто → KYC_REQUIRED_DOC_TYPES
}

type HasRequiredDocumentsOutput struct {
	HasAll  bool
	Missing []domain.Type
}

func (uc *HasRequiredDocumentsUseCase) HasRequiredDocuments(ctx context.Context, in HasRequiredDocumentsInput) (HasRequiredDocumentsOutput, error) {
	if err := requireAccess(ctx, in.CustomerID); err != nil {
		return HasRequiredDocumentsOutput{}, err
	}

	required := in.Required
	if len(required) == 0 {
		required = uc.defaultRequired
	}
	for _, t := range required {
		if !t.Valid() {
			return HasRequiredDocumentsOutput{}, domain.ErrInvalidDocumentType
		}
	}

	slog.Info("HasRequiredDocuments check", "customer_id", in.CustomerID.String(), "required", required)
	confirmed, err := uc.repo.ConfirmedTypes(ctx, in.CustomerID)
	if err != nil {
		return HasRequiredDocumentsOutput{}, err
	}
	slog.Info("HasRequiredDocuments result", "confirmed", confirmed)
	missing := domain.Missing(required, confirmed)
	return HasRequiredDocumentsOutput{HasAll: len(missing) == 0, Missing: missing}, nil
}
