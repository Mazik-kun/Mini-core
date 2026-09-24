package main

import (
	"context"
	"errors"
	"log/slog"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/postgres/db"
	"github.com/Mazik-kun/mini-core/services/auth/internal/domain/password"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authServer struct {
	authv1.UnimplementedAuthServiceServer
	log     *slog.Logger
	queries *db.Queries
}

func newAuthServer(log *slog.Logger, queries *db.Queries) *authServer {
	return &authServer{log: log, queries: queries}
}

func (s *authServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	panic("test panic")
	email := req.GetEmail()
	s.log.Info("register called", "email", email)
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	pass := req.GetPassword()
	if pass == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	passwordHash, err := password.HashPassword(pass)

	if err != nil {
		s.log.Error("failed to hash paasword", "err", err)
		return nil, status.Error(codes.Internal, "failed to hash password")
	}
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         "CLIENT",
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			s.log.Info("email already existrs", "email", email)
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		s.log.Error("create user failed", "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &authv1.RegisterResponse{
		UserId: user.ID.String(),
		Email:  user.Email,
	}, nil

}
func (s *authServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	s.log.Info("login called", "email", req.GetEmail())

	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	return &authv1.LoginResponse{
		AccessToken:  "fake-acces-token",
		RefreshToken: "fake-refresh-token",
		ExpiresIn:    900,
	}, nil
}
