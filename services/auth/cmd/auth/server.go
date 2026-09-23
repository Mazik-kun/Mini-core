package main

import (
	"context"
	"log/slog"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/postgres/db"
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
	s.log.Info("register called", "email", req.GetEmail())

	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	return &authv1.RegisterResponse{
		UserId: "00000000-0000-0000-0000-000000000000",
		Email:  req.GetEmail(),
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
