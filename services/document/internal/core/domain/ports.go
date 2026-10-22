package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrDocumentNotConfirmed = errors.New("document is not confirmed")

// ErrObjectNotFound возвращает ObjectStorage, если объекта нет в хранилище.
var ErrObjectNotFound = errors.New("object not found in storage")

type ListFilter struct {
	CustomerID uuid.UUID
	Status     *Status
	Cursor     uuid.UUID // uuid.Nil = с начала
	Limit      int
}

type DocumentRepository interface {
	Create(ctx context.Context, doc Document) (Document, error)
	GetByID(ctx context.Context, id uuid.UUID) (Document, error)
	// MarkConfirmed: PENDING → CONFIRMED. Если документ не в PENDING — ErrInvalidStatusTransition.
	MarkConfirmed(ctx context.Context, id uuid.UUID, contentType string, sizeBytes int64, sha256Hex string) (Document, error)
	// MarkRejected: PENDING → REJECTED. Если документ не в PENDING — ErrInvalidStatusTransition.
	MarkRejected(ctx context.Context, id uuid.UUID, contentType string, sizeBytes int64) (Document, error)
	List(ctx context.Context, filter ListFilter) ([]Document, error)
	ConfirmedTypes(ctx context.Context, customerID uuid.UUID) ([]Type, error)
}

type ObjectInfo struct {
	SizeBytes int64
}

type ObjectStorage interface {
	PresignUpload(ctx context.Context, key string, ttl time.Duration) (string, error)
	PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error) // ErrObjectNotFound
	ReadHead(ctx context.Context, key string, n int) ([]byte, error)
	SHA256(ctx context.Context, key string) (string, error) // hex, стримом
	Remove(ctx context.Context, key string) error
}
