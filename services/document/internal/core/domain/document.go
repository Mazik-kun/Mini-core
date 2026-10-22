package domain

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type (
	Type   string
	Status string
)

const (
	TypePassport       Type = "PASSPORT"
	TypeSelfie         Type = "SELFIE"
	TypeProofOfAddress Type = "PROOF_OF_ADDRESS"

	StatusPending   Status = "PENDING"
	StatusConfirmed Status = "CONFIRMED"
	StatusRejected  Status = "REJECTED"
)

var (
	ErrDocumentNotFound        = errors.New("document not found")
	ErrAccessDenied            = errors.New("access denied")
	ErrInvalidDocumentType     = errors.New("invalid document type")
	ErrInvalidCustomerID       = errors.New("invalid customer_id")
	ErrInvalidDocumentID       = errors.New("invalid document_id")
	ErrFileTooLarge            = errors.New("file exceeds max size")
	ErrUnsupportedContentType  = errors.New("unsupported content type")
	ErrObjectNotUploaded       = errors.New("object is not uploaded yet")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidPageToken        = errors.New("invalid page_token")
)

func (t Type) Valid() bool {
	switch t {
	case TypePassport, TypeSelfie, TypeProofOfAddress:
		return true
	}
	return false
}

// ParseType разбирает строку (используется для KYC_REQUIRED_DOC_TYPES).
func ParseType(s string) (Type, error) {
	t := Type(strings.ToUpper(strings.TrimSpace(s)))
	if !t.Valid() {
		return "", ErrInvalidDocumentType
	}
	return t, nil
}

var allowedTransitions = map[Status][]Status{
	StatusPending:   {StatusConfirmed, StatusRejected},
	StatusConfirmed: nil,
	StatusRejected:  nil,
}

func (s Status) CanTransitionTo(next Status) bool {
	for _, a := range allowedTransitions[s] {
		if a == next {
			return true
		}
	}
	return false
}

// AllowedContentTypes — допустимые типы по magic bytes.
var AllowedContentTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"application/pdf": {},
}

// DetectContentType определяет тип по первым ≤512 байтам и проверяет белый список.
func DetectContentType(head []byte) (string, error) {
	ct := http.DetectContentType(head)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	if _, ok := AllowedContentTypes[ct]; !ok {
		return ct, ErrUnsupportedContentType
	}
	return ct, nil
}

// ObjectKey: {customer_id}/{document_type}/{document_id}.
func ObjectKey(customerID uuid.UUID, t Type, documentID uuid.UUID) string {
	return customerID.String() + "/" + string(t) + "/" + documentID.String()
}

type Document struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	Type        Type
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	SHA256      string
	Status      Status
	CreatedAt   time.Time
	ConfirmedAt *time.Time
}

// Missing возвращает обязательные типы, по которым нет CONFIRMED-документа.
func Missing(required []Type, confirmed []Type) []Type {
	have := make(map[Type]struct{}, len(confirmed))
	for _, t := range confirmed {
		have[t] = struct{}{}
	}
	var out []Type
	for _, t := range required {
		if _, ok := have[t]; !ok {
			out = append(out, t)
		}
	}
	return out
}
