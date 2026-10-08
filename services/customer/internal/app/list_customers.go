package app

import (
	"context"
	"encoding/base64"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

type ListCustomersUseCase struct {
	repo domain.CustomerRepository
}

func NewListCustomersUseCase(repo domain.CustomerRepository) *ListCustomersUseCase {
	return &ListCustomersUseCase{repo: repo}
}

type ListCustomersInput struct {
	RequesterRole string
	Statuses      []domain.Status
	PageSize      int
	PageToken     string
}

type ListCustomersOutput struct {
	Customers     []domain.Customer
	NextPageToken string // пустая строка — страниц больше нет
}

func (uc *ListCustomersUseCase) ListCustomers(ctx context.Context, in ListCustomersInput) (ListCustomersOutput, error) {
	if in.RequesterRole != domain.RoleOfficer && in.RequesterRole != domain.RoleAdmin {
		return ListCustomersOutput{}, domain.ErrAccessDenied
	}

	for _, status := range in.Statuses {
		if !status.IsValid() {
			return ListCustomersOutput{}, domain.ErrInvalidStatus
		}
	}

	size := in.PageSize
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}

	cursor, err := decodePageToken(in.PageToken)
	if err != nil {
		return ListCustomersOutput{}, err
	}

	// Пустой фильтр передаём как nil: в SQL "statuses IS NULL" означает "без фильтра".
	var statuses []domain.Status
	if len(in.Statuses) > 0 {
		statuses = in.Statuses
	}

	// Просим на одну запись больше, чтобы узнать, есть ли следующая страница.
	customers, err := uc.repo.List(ctx, domain.ListFilter{
		Statuses: statuses,
		Limit:    size + 1,
		Cursor:   cursor,
	})
	if err != nil {
		return ListCustomersOutput{}, err
	}

	out := ListCustomersOutput{Customers: customers}
	if len(customers) > size {
		out.Customers = customers[:size]
		out.NextPageToken = encodePageToken(customers[size-1].ID)
	}
	return out, nil
}

// Токен — непрозрачная для клиента строка: base64 от 16 байт UUID последней записи страницы.
func encodePageToken(id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString(id[:])
}

func decodePageToken(token string) (uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return uuid.Nil, domain.ErrInvalidPageToken
	}
	var id uuid.UUID
	if len(raw) != 16 {
		return uuid.Nil, domain.ErrInvalidPageToken
	}
	copy(id[:], raw)
	return id, nil
}