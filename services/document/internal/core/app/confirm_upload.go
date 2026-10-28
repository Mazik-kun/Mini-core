package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

const headSize = 512

type ConfirmUploadUseCase struct {
	repo    domain.DocumentRepository
	storage domain.ObjectStorage
	maxSize int64
}

func NewConfirmUploadUseCase(repo domain.DocumentRepository, storage domain.ObjectStorage, maxSize int64) *ConfirmUploadUseCase {
	return &ConfirmUploadUseCase{repo: repo, storage: storage, maxSize: maxSize}
}

type ConfirmUploadInput struct {
	DocumentID uuid.UUID
}

func (uc *ConfirmUploadUseCase) ConfirmUpload(ctx context.Context, in ConfirmUploadInput) (domain.Document, error) {
	doc, err := uc.repo.GetByID(ctx, in.DocumentID)
	if err != nil {
		return domain.Document{}, err
	}
	if err := requireAccess(ctx, doc.CustomerID); err != nil {
		return domain.Document{}, err
	}

	switch doc.Status {
	case domain.StatusConfirmed:
		return doc, nil // идемпотентность: без повторной валидации, MinIO не трогаем
	case domain.StatusRejected:
		return domain.Document{}, domain.ErrInvalidStatusTransition
	}

	info, err := uc.storage.Stat(ctx, doc.ObjectKey)
	if err != nil {
		if errors.Is(err, domain.ErrObjectNotFound) {
			return domain.Document{}, domain.ErrObjectNotUploaded
		}
		return domain.Document{}, fmt.Errorf("stat object: %w", err)
	}
	if info.SizeBytes > uc.maxSize {
		return domain.Document{}, uc.reject(ctx, doc, "", info.SizeBytes, domain.ErrFileTooLarge)
	}

	readSize := headSize
	if info.SizeBytes < int64(headSize) {
		readSize = int(info.SizeBytes)
	}

	head, err := uc.storage.ReadHead(ctx, doc.ObjectKey, readSize)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read object head: %w", err)
	}

	if err != nil {
		return domain.Document{}, fmt.Errorf("read object head: %w", err)
	}
	contentType, err := domain.DetectContentType(head)
	if err != nil {
		return domain.Document{}, uc.reject(ctx, doc, contentType, info.SizeBytes, err)
	}

	sum, err := uc.storage.SHA256(ctx, doc.ObjectKey)
	if err != nil {
		return domain.Document{}, fmt.Errorf("hash object: %w", err)
	}

	confirmed, err := uc.repo.MarkConfirmed(ctx, doc.ID, contentType, info.SizeBytes, sum)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidStatusTransition) {
			// гонка с параллельным Confirm: если уже CONFIRMED — вернуть как есть
			current, getErr := uc.repo.GetByID(ctx, doc.ID)
			if getErr == nil && current.Status == domain.StatusConfirmed {
				return current, nil
			}
		}
		return domain.Document{}, err
	}
	return confirmed, nil
}

func (uc *ConfirmUploadUseCase) reject(ctx context.Context, doc domain.Document, contentType string, size int64, cause error) error {
	if _, err := uc.repo.MarkRejected(ctx, doc.ID, contentType, size); err != nil {
		if !errors.Is(err, domain.ErrInvalidStatusTransition) {
			return err
		}
	}
	if err := uc.storage.Remove(ctx, doc.ObjectKey); err != nil {
		return fmt.Errorf("remove rejected object: %w", err)
	}
	return cause
}
