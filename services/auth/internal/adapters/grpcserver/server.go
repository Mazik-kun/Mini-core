package grpcserver

import (
	"context"
	"log/slog"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/services/auth/internal/app"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	register *app.RegisterUseCase
	login    *app.LoginUseCase
	log      *slog.Logger
}

func NewAuthServer(register *app.RegisterUseCase, login *app.LoginUseCase, log *slog.Logger) *AuthServer {
	return &AuthServer{register: register, login: login, log: log}
}

func (s *AuthServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	s.log.Info("register called", "email", req.GetEmail())

	out, err := s.register.Register(ctx, app.RegisterInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.RegisterResponse{
		UserId: out.UserID.String(),
		Email:  out.Email,
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	s.log.Info("login called", "email", req.GetEmail())

	_, err := s.login.Login(ctx, app.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	// TODO: настоящие JWT-токены
	return &authv1.LoginResponse{
		AccessToken:  "fake-access-token",
		RefreshToken: "fake-refresh-token",
		ExpiresIn:    900,
	}, nil
}