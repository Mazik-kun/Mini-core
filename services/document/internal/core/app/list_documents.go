package app

import (
	"context"
	"encoding/base64"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

type ListDocumentsUseCase struct {
	repo        domain.DocumentRepository
	defaultSize int
	maxSize     int
}

func NewListDocumentsUseCase(repo domain.DocumentRepository, defaultSize, maxSize int) *ListDocumentsUseCase {
	return &ListDocumentsUseCase{repo: repo, defaultSize: defaultSize, maxSize: maxSize}
}

type ListDocumentsInput struct {
	CustomerID uuid.UUID
	Status     *domain.Status
	PageSize   int
	PageToken  string
}

type ListDocumentsOutput struct {
	Documents     []domain.Document
	NextPageToken string
}

func (uc *ListDocumentsUseCase) ListDocuments(ctx context.Context, in ListDocumentsInput) (ListDocumentsOutput, error) {
	if err := requireAccess(ctx, in.CustomerID); err != nil {
		return ListDocumentsOutput{}, err
	}

	cursor, err := decodePageToken(in.PageToken)
	if err != nil {
		return ListDocumentsOutput{}, err
	}

	limit := in.PageSize
	if limit <= 0 {
		limit = uc.defaultSize
	}
	if limit > uc.maxSize {
		limit = uc.maxSize
	}

	docs, err := uc.repo.List(ctx, domain.ListFilter{
		CustomerID: in.CustomerID,
		Status:     in.Status,
		Cursor:     cursor,
		Limit:      limit + 1, // +1 — признак следующей страницы
	})
	if err != nil {
		return ListDocumentsOutput{}, err
	}

	out := ListDocumentsOutput{Documents: docs}
	if len(docs) > limit {
		out.Documents = docs[:limit]
		out.NextPageToken = encodePageToken(docs[limit-1].ID)
	}
	return out, nil
}

func encodePageToken(id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id.String()))
}

func decodePageToken(token string) (uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return uuid.Nil, domain.ErrInvalidPageToken
	}
	id, err := uuid.ParseBytes(raw)
	if err != nil {
		return uuid.Nil, domain.ErrInvalidPageToken
	}
	return id, nil
}
