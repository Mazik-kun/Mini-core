package grpcserver

import (
	"errors"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCustomerID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrCustomerNotFound):
		return status.Error(codes.Unknown, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
	
}