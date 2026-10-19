package grpcserver

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Mazik-kun/mini-core/services/auth/internal/core/domain"
)

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrWeakPassword):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrInvalidToken),
		errors.Is(err, domain.ErrTokenExpired),
		errors.Is(err, domain.ErrRefreshTokenNotFound):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrPermissionDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
