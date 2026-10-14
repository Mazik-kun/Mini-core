package grpcserver

import (
	"context"
	"log/slog"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/services/auth/internal/app"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	register      *app.RegisterUseCase
	login         *app.LoginUseCase
	refresh       *app.RefreshUseCase
	logout        *app.LogoutUseCase
	validateToken *app.ValidateTokenUseCase
	createOfficer *app.CreateOfficerUseCase
	log           *slog.Logger
}

func NewAuthServer(
	register *app.RegisterUseCase,
	login *app.LoginUseCase,
	refresh *app.RefreshUseCase,
	logout *app.LogoutUseCase,
	validateToken *app.ValidateTokenUseCase,
	createOfficer *app.CreateOfficerUseCase,
	log *slog.Logger,
) *AuthServer {
	return &AuthServer{
		register:      register,
		login:         login,
		refresh:       refresh,
		logout:        logout,
		validateToken: validateToken,
		createOfficer: createOfficer,
		log:           log,
	}
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
		Role:   authv1.Role_ROLE_CLIENT,
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	s.log.Info("login called", "email", req.GetEmail())

	out, err := s.login.Login(ctx, app.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.LoginResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresIn:    out.ExpiresIn,
	}, nil
}

func (s *AuthServer) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.LoginResponse, error) {
	out, err := s.refresh.Refresh(ctx, app.RefreshInput{
		RefreshToken: req.GetRefreshToken(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &authv1.LoginResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresIn:    out.ExpiresIn,
	}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*emptypb.Empty, error) {
	err := s.logout.Logout(ctx, app.LogoutInput{
		RefreshToken: req.GetRefreshToken(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *AuthServer) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	out, err := s.validateToken.ValidateToken(ctx, app.ValidateTokenInput{
		Token: req.GetAccessToken(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	resp := &authv1.ValidateTokenResponse{
		UserId: out.Claims.UserID.String(),
		Role:   roleToProto(out.Claims.Role),
	}
	if out.Claims.CustomerID != nil {
		resp.CustomerId = out.Claims.CustomerID.String()
	}
	return resp, nil
}

func (s *AuthServer) CreateOfficer(ctx context.Context, req *authv1.CreateOfficerRequest) (*authv1.RegisterResponse, error) {
	s.log.Info("create officer called", "email", req.GetEmail())

	out, err := s.createOfficer.CreateOfficer(ctx, app.CreateOfficerInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.RegisterResponse{
		UserId: out.UserID.String(),
		Email:  out.Email,
		Role:   authv1.Role_ROLE_OFFICER,
	}, nil
}