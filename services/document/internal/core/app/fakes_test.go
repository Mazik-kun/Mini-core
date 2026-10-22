package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

func clientCtx(customerID uuid.UUID) context.Context {
	return authjwt.WithClaims(context.Background(), authjwt.Claims{
		Subject:    uuid.New(),
		Role:       roles.RoleClient,
		CustomerID: &customerID,
	})
}

func officerCtx() context.Context {
	return authjwt.WithClaims(context.Background(), authjwt.Claims{
		Subject: uuid.New(),
		Role:    roles.RoleOfficer,
	})
}

type fakeRepo struct {
	docs map[uuid.UUID]domain.Document
}

func newFakeRepo() *fakeRepo { return &fakeRepo{docs: map[uuid.UUID]domain.Document{}} }

func (r *fakeRepo) Create(_ context.Context, d domain.Document) (domain.Document, error) {
	d.CreatedAt = time.Now().UTC()
	r.docs[d.ID] = d
	return d, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Document, error) {
	d, ok := r.docs[id]
	if !ok {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return d, nil
}

func (r *fakeRepo) MarkConfirmed(_ context.Context, id uuid.UUID, ct string, size int64, sum string) (domain.Document, error) {
	d, ok := r.docs[id]
	if !ok {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if !d.Status.CanTransitionTo(domain.StatusConfirmed) {
		return domain.Document{}, domain.ErrInvalidStatusTransition
	}
	now := time.Now().UTC()
	d.Status, d.ContentType, d.SizeBytes, d.SHA256, d.ConfirmedAt = domain.StatusConfirmed, ct, size, sum, &now
	r.docs[id] = d
	return d, nil
}

func (r *fakeRepo) MarkRejected(_ context.Context, id uuid.UUID, ct string, size int64) (domain.Document, error) {
	d, ok := r.docs[id]
	if !ok {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if !d.Status.CanTransitionTo(domain.StatusRejected) {
		return domain.Document{}, domain.ErrInvalidStatusTransition
	}
	d.Status, d.ContentType, d.SizeBytes = domain.StatusRejected, ct, size
	r.docs[id] = d
	return d, nil
}

func (r *fakeRepo) List(_ context.Context, f domain.ListFilter) ([]domain.Document, error) {
	var out []domain.Document
	for _, d := range r.docs {
		if d.CustomerID != f.CustomerID || (f.Status != nil && d.Status != *f.Status) {
			continue
		}
		if f.Cursor != uuid.Nil && d.ID.String() >= f.Cursor.String() {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID.String() > out[j].ID.String() })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (r *fakeRepo) ConfirmedTypes(_ context.Context, customerID uuid.UUID) ([]domain.Type, error) {
	seen := map[domain.Type]struct{}{}
	var out []domain.Type
	for _, d := range r.docs {
		if d.CustomerID == customerID && d.Status == domain.StatusConfirmed {
			if _, ok := seen[d.Type]; !ok {
				seen[d.Type] = struct{}{}
				out = append(out, d.Type)
			}
		}
	}
	return out, nil
}

func (r *fakeRepo) seed(d domain.Document) { r.docs[d.ID] = d }

// ---- fake storage ----

type fakeStorage struct {
	objects   map[string][]byte
	removed   []string
	statCalls int
}

func newFakeStorage() *fakeStorage { return &fakeStorage{objects: map[string][]byte{}} }

func (s *fakeStorage) PresignUpload(_ context.Context, key string, _ time.Duration) (string, error) {
	return "http://minio.test/put/" + key, nil
}

func (s *fakeStorage) PresignDownload(_ context.Context, key string, _ time.Duration) (string, error) {
	return "http://minio.test/get/" + key, nil
}

func (s *fakeStorage) Stat(_ context.Context, key string) (domain.ObjectInfo, error) {
	s.statCalls++
	b, ok := s.objects[key]
	if !ok {
		return domain.ObjectInfo{}, domain.ErrObjectNotFound
	}
	return domain.ObjectInfo{SizeBytes: int64(len(b))}, nil
}

func (s *fakeStorage) ReadHead(_ context.Context, key string, n int) ([]byte, error) {
	b, ok := s.objects[key]
	if !ok {
		return nil, domain.ErrObjectNotFound
	}
	if len(b) > n {
		b = b[:n]
	}
	return b, nil
}

func (s *fakeStorage) SHA256(_ context.Context, key string) (string, error) {
	b, ok := s.objects[key]
	if !ok {
		return "", domain.ErrObjectNotFound
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (s *fakeStorage) Remove(_ context.Context, key string) error {
	delete(s.objects, key)
	s.removed = append(s.removed, key)
	return nil
}

// ---- fixtures ----

func pngBytes(extra int) []byte {
	return append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, extra)...)
}

func pendingDoc(customerID uuid.UUID, t domain.Type) domain.Document {
	id, _ := uuid.NewV7()
	return domain.Document{
		ID: id, CustomerID: customerID, Type: t,
		ObjectKey: domain.ObjectKey(customerID, t, id), Status: domain.StatusPending,
	}
}
