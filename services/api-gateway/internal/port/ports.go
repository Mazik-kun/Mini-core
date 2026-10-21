// Package port — интерфейсы, которые потребляют middleware и хендлеры.
package port

import (
	"context"
	"crypto/ed25519"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
)

var (
	// ErrKeyNotFound — kid неизвестен даже после on-demand refresh.
	ErrKeyNotFound = errors.New("jwks: key not found")
	// ErrJWKSUnavailable — не удалось получить JWKS у auth-service.
	ErrJWKSUnavailable = errors.New("jwks: unavailable")
)

type JWKSProvider interface {
	KeyByID(ctx context.Context, kid string) (ed25519.PublicKey, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, key string) (allowed bool, retryAfter time.Duration, err error)
}

// AuthClient — интерфейс для хендлеров auth.
type AuthClient interface {
	Register(ctx context.Context, in *authv1.RegisterRequest, opts ...grpc.CallOption) (*authv1.RegisterResponse, error)
	Login(ctx context.Context, in *authv1.LoginRequest, opts ...grpc.CallOption) (*authv1.LoginResponse, error)
	Refresh(ctx context.Context, in *authv1.RefreshRequest, opts ...grpc.CallOption) (*authv1.LoginResponse, error)
	Logout(ctx context.Context, in *authv1.LogoutRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	CreateOfficer(ctx context.Context, in *authv1.CreateOfficerRequest, opts ...grpc.CallOption) (*authv1.RegisterResponse, error)
}

// CustomerClient — часть customerv1.CustomerServiceClient, нужная ручным хендлерам.
type CustomerClient interface {
	GetCustomer(ctx context.Context, in *customerv1.GetCustomerRequest, opts ...grpc.CallOption) (*customerv1.Customer, error)
	UpdateProfile(ctx context.Context, in *customerv1.UpdateProfileRequest, opts ...grpc.CallOption) (*customerv1.Customer, error)
	GetCustomerStatus(ctx context.Context, in *customerv1.GetCustomerStatusRequest, opts ...grpc.CallOption) (*customerv1.GetCustomerStatusResponse, error)
	ListCustomers(ctx context.Context, in *customerv1.ListCustomersRequest, opts ...grpc.CallOption) (*customerv1.ListCustomersResponse, error)
}