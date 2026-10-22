package grpcserver

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	documentv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/document/v1"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

func typeFromProto(t documentv1.DocumentType) (domain.Type, bool) {
	switch t {
	case documentv1.DocumentType_DOCUMENT_TYPE_PASSPORT:
		return domain.TypePassport, true
	case documentv1.DocumentType_DOCUMENT_TYPE_SELFIE:
		return domain.TypeSelfie, true
	case documentv1.DocumentType_DOCUMENT_TYPE_PROOF_OF_ADDRESS:
		return domain.TypeProofOfAddress, true
	default:
		return "", false
	}
}

func typeToProto(t domain.Type) documentv1.DocumentType {
	switch t {
	case domain.TypePassport:
		return documentv1.DocumentType_DOCUMENT_TYPE_PASSPORT
	case domain.TypeSelfie:
		return documentv1.DocumentType_DOCUMENT_TYPE_SELFIE
	case domain.TypeProofOfAddress:
		return documentv1.DocumentType_DOCUMENT_TYPE_PROOF_OF_ADDRESS
	default:
		return documentv1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED
	}
}

func statusToProto(s domain.Status) documentv1.DocumentStatus {
	switch s {
	case domain.StatusPending:
		return documentv1.DocumentStatus_DOCUMENT_STATUS_PENDING
	case domain.StatusConfirmed:
		return documentv1.DocumentStatus_DOCUMENT_STATUS_CONFIRMED
	case domain.StatusRejected:
		return documentv1.DocumentStatus_DOCUMENT_STATUS_REJECTED
	default:
		return documentv1.DocumentStatus_DOCUMENT_STATUS_UNSPECIFIED
	}
}

// statusFilterFromProto: UNSPECIFIED → nil (без фильтра). ok=false — недопустимое значение.
func statusFilterFromProto(s documentv1.DocumentStatus) (*domain.Status, bool) {
	var out domain.Status
	switch s {
	case documentv1.DocumentStatus_DOCUMENT_STATUS_UNSPECIFIED:
		return nil, true
	case documentv1.DocumentStatus_DOCUMENT_STATUS_PENDING:
		out = domain.StatusPending
	case documentv1.DocumentStatus_DOCUMENT_STATUS_CONFIRMED:
		out = domain.StatusConfirmed
	case documentv1.DocumentStatus_DOCUMENT_STATUS_REJECTED:
		out = domain.StatusRejected
	default:
		return nil, false
	}
	return &out, true
}

func documentToProto(d domain.Document) *documentv1.Document {
	return &documentv1.Document{
		DocumentId:   d.ID.String(),
		CustomerId:   d.CustomerID.String(),
		DocumentType: typeToProto(d.Type),
		ContentType:  d.ContentType,
		SizeBytes:    d.SizeBytes,
		Status:       statusToProto(d.Status),
		Sha256:       d.SHA256,
		CreatedAt:    timestamppb.New(d.CreatedAt),
	}
}
