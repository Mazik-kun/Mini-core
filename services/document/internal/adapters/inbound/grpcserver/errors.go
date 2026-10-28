package grpcserver

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrDocumentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrInvalidStatusTransition),
		errors.Is(err, domain.ErrObjectNotUploaded),
		errors.Is(err, domain.ErrDocumentNotConfirmed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidDocumentType),
		errors.Is(err, domain.ErrInvalidCustomerID),
		errors.Is(err, domain.ErrInvalidDocumentID),
		errors.Is(err, domain.ErrFileTooLarge),
		errors.Is(err, domain.ErrUnsupportedContentType),
		errors.Is(err, domain.ErrInvalidPageToken):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
