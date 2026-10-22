package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Mazik-kun/mini-core/services/document/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

type DocumentRepository struct {
	q *db.Queries
}

func NewDocumentRepository(q *db.Queries) *DocumentRepository {
	return &DocumentRepository{q: q}
}

// ---------- Create ----------

func (r *DocumentRepository) Create(ctx context.Context, doc domain.Document) (domain.Document, error) {
	row, err := r.q.CreateDocument(ctx, db.CreateDocumentParams{
		ID:           doc.ID,
		CustomerID:   doc.CustomerID,
		DocumentType: string(doc.Type),
		ObjectKey:    doc.ObjectKey,
		Status:       string(doc.Status),
	})
	if err != nil {
		return domain.Document{}, fmt.Errorf("create document: %w", err)
	}
	return toDomain(row), nil
}

// ---------- GetByID ----------

func (r *DocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Document, error) {
	row, err := r.q.GetDocumentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Document{}, domain.ErrDocumentNotFound
		}
		return domain.Document{}, fmt.Errorf("get document by id: %w", err)
	}
	return toDomain(row), nil
}

// ---------- MarkConfirmed ----------

func (r *DocumentRepository) MarkConfirmed(ctx context.Context, id uuid.UUID, contentType string, sizeBytes int64, sha256Hex string) (domain.Document, error) {
	row, err := r.q.ConfirmDocument(ctx, db.ConfirmDocumentParams{
		ID:          id,
		ContentType: pgtype.Text{String: contentType, Valid: true},
		SizeBytes:   pgtype.Int8{Int64: sizeBytes, Valid: true},
		Sha256:      pgtype.Text{String: sha256Hex, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// условный UPDATE (status='PENDING') ничего не изменил
			return domain.Document{}, domain.ErrInvalidStatusTransition
		}
		return domain.Document{}, fmt.Errorf("confirm document: %w", err)
	}
	return toDomain(row), nil
}

// ---------- MarkRejected ----------

func (r *DocumentRepository) MarkRejected(ctx context.Context, id uuid.UUID, contentType string, sizeBytes int64) (domain.Document, error) {
	row, err := r.q.RejectDocument(ctx, db.RejectDocumentParams{
		ID:          id,
		ContentType: pgtype.Text{String: contentType, Valid: contentType != ""},
		SizeBytes:   pgtype.Int8{Int64: sizeBytes, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Document{}, domain.ErrInvalidStatusTransition
		}
		return domain.Document{}, fmt.Errorf("reject document: %w", err)
	}
	return toDomain(row), nil
}

// ---------- List ----------

func (r *DocumentRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Document, error) {
	params := db.ListDocumentsParams{
		CustomerID: filter.CustomerID,
		Lim:        int32(filter.Limit),
	}
	if filter.Status != nil {
		params.Status = pgtype.Text{String: string(*filter.Status), Valid: true}
	}
	if filter.Cursor != uuid.Nil {
		params.Cursor = pgtype.UUID{Bytes: filter.Cursor, Valid: true}
	}

	rows, err := r.q.ListDocuments(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	out := make([]domain.Document, len(rows))
	for i, row := range rows {
		out[i] = toDomain(row)
	}
	return out, nil
}

// ---------- ConfirmedTypes ----------

func (r *DocumentRepository) ConfirmedTypes(ctx context.Context, customerID uuid.UUID) ([]domain.Type, error) {
	rows, err := r.q.ListConfirmedDocumentTypes(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list confirmed document types: %w", err)
	}
	out := make([]domain.Type, len(rows))
	for i, t := range rows {
		out[i] = domain.Type(t)
	}
	return out, nil
}

// ---------- mapper ----------

func toDomain(row db.DocumentDocument) domain.Document {
	doc := domain.Document{
		ID:          row.ID,
		CustomerID:  row.CustomerID,
		Type:        domain.Type(row.DocumentType),
		ObjectKey:   row.ObjectKey,
		ContentType: row.ContentType.String,
		SizeBytes:   row.SizeBytes.Int64,
		SHA256:      row.Sha256.String,
		Status:      domain.Status(row.Status),
		CreatedAt:   row.CreatedAt.Time.UTC(),
	}
	if row.ConfirmedAt.Valid {
		t := row.ConfirmedAt.Time.UTC()
		doc.ConfirmedAt = &t
	}
	return doc
}
