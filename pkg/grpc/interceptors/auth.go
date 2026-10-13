package interceptors

import (
	"context"
	"strings"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// bearerPrefix — префикс схемы авторизации в заголовке authorization.
const bearerPrefix = "Bearer "

// UnaryAuthInterceptor проверяет JWT из metadata authorization и
// кладёт Claims в context. Методы из publicMethods пропускаются без токена.
func UnaryAuthInterceptor(
	verifier *authjwt.Verifier,
	publicMethods map[string]bool,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		authorizationHeader, err := authorizationFromIncomingMetadata(ctx)
		if err != nil {
			return nil, err
		}

		tokenString, err := tokenFromAuthorizationHeader(authorizationHeader)
		if err != nil {
			return nil, err
		}

		claims, err := verifier.VerifyToken(tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid access token")
		}

		ctx = authjwt.WithClaims(ctx, claims)
		return handler(ctx, req)
	}
}

// authorizationFromIncomingMetadata достаёт значение заголовка authorization
// из входящих gRPC-метаданных.
func authorizationFromIncomingMetadata(ctx context.Context) (string, error) {
	incomingMetadata, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	authorizationValues := incomingMetadata.Get("authorization")
	if len(authorizationValues) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	return authorizationValues[0], nil
}

// tokenFromAuthorizationHeader отрезает префикс "Bearer " и возвращает токен.
func tokenFromAuthorizationHeader(authorizationHeader string) (string, error) {
	tokenString, found := strings.CutPrefix(authorizationHeader, bearerPrefix)
	if !found || tokenString == "" {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header")
	}
	return tokenString, nil
}