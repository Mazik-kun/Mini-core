package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

type GetDownloadURLUseCase struct {
	repo    domain.DocumentRepository
	storage domain.ObjectStorage
	ttl     time.Duration
}

func NewGetDownloadURLUseCase(repo domain.DocumentRepository, storage domain.ObjectStorage, ttl time.Duration) *GetDownloadURLUseCase {
	return &GetDownloadURLUseCase{repo: repo, storage: storage, ttl: ttl}
}

type GetDownloadURLInput struct {
	DocumentID uuid.UUID
}

type GetDownloadURLOutput struct {
	URL       string
	ExpiresAt time.Time
}

func (uc *GetDownloadURLUseCase) GetDownloadURL(ctx context.Context, in GetDownloadURLInput) (GetDownloadURLOutput, error) {
	doc, err := uc.repo.GetByID(ctx, in.DocumentID)
	if err != nil {
		return GetDownloadURLOutput{}, err
	}
	if err := requireAccess(ctx, doc.CustomerID); err != nil {
		return GetDownloadURLOutput{}, err
	}
	if doc.Status != domain.StatusConfirmed {
		return GetDownloadURLOutput{}, domain.ErrDocumentNotConfirmed
	}

	expiresAt := time.Now().UTC().Add(uc.ttl)
	url, err := uc.storage.PresignDownload(ctx, doc.ObjectKey, uc.ttl)
	if err != nil {
		return GetDownloadURLOutput{}, fmt.Errorf("presign download: %w", err)
	}
	return GetDownloadURLOutput{URL: url, ExpiresAt: expiresAt}, nil
}
