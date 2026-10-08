package grpcserver

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrCustomerNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrProfileLocked),
		errors.Is(err, domain.ErrInvalidStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidCustomerID),
		errors.Is(err, domain.ErrInvalidFullName),
		errors.Is(err, domain.ErrInvalidBirthDate),
		errors.Is(err, domain.ErrInvalidAddress),
		errors.Is(err, domain.ErrInvalidPhoneNumber),
		errors.Is(err, domain.ErrInvalidCitizenship),
		errors.Is(err, domain.ErrInvalidStatus),
		errors.Is(err, domain.ErrInvalidPageToken):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}