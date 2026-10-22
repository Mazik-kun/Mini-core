package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

type InitUploadUseCase struct {
	repo      domain.DocumentRepository
	storage   domain.ObjectStorage
	uploadTTL time.Duration
}

func NewInitUploadUseCase(repo domain.DocumentRepository, storage domain.ObjectStorage, uploadTTL time.Duration) *InitUploadUseCase {
	return &InitUploadUseCase{repo: repo, storage: storage, uploadTTL: uploadTTL}
}

type InitUploadInput struct {
	CustomerID uuid.UUID
	Type       domain.Type
}

type InitUploadOutput struct {
	DocumentID uuid.UUID
	UploadURL  string
}

func (uc *InitUploadUseCase) InitUpload(ctx context.Context, in InitUploadInput) (InitUploadOutput, error) {
	if err := requireAccess(ctx, in.CustomerID); err != nil {
		return InitUploadOutput{}, err
	}
	if !in.Type.Valid() {
		return InitUploadOutput{}, domain.ErrInvalidDocumentType
	}

	documentID, err := uuid.NewV7()
	if err != nil {
		return InitUploadOutput{}, fmt.Errorf("generate document id: %w", err)
	}
	key := domain.ObjectKey(in.CustomerID, in.Type, documentID)

	url, err := uc.storage.PresignUpload(ctx, key, uc.uploadTTL)
	if err != nil {
		return InitUploadOutput{}, fmt.Errorf("presign upload: %w", err)
	}

	if _, err := uc.repo.Create(ctx, domain.Document{
		ID:         documentID,
		CustomerID: in.CustomerID,
		Type:       in.Type,
		ObjectKey:  key,
		Status:     domain.StatusPending,
	}); err != nil {
		return InitUploadOutput{}, err
	}

	return InitUploadOutput{DocumentID: documentID, UploadURL: url}, nil
}
