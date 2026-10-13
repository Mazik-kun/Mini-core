package interceptors

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// testVerifier создаёт Verifier с новой парой ключей Ed25519.
func testVerifier(t *testing.T) (*authjwt.Verifier, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	return authjwt.NewVerifier(publicKey), privateKey
}

// signAccessToken выпускает JWT, подписанный переданным приватным ключом.
func signAccessToken(t *testing.T, privateKey ed25519.PrivateKey, userID uuid.UUID, role string) string {
	t.Helper()
	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub":  userID.String(),
		"role": role,
		"exp":  now.Add(time.Hour).Unix(),
		"iat":  now.Unix(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signedToken
}

// passThroughHandler — простой хендлер, который ничего не проверяет.
func passThroughHandler(ctx context.Context, req any) (any, error) {
	return "ok", nil
}

// contextCapturingHandler запоминает context, чтобы тест мог достать Claims.
type contextCapturingHandler struct {
	capturedContext context.Context
}

func (h *contextCapturingHandler) Handle(ctx context.Context, req any) (any, error) {
	h.capturedContext = ctx
	return "ok", nil
}

func TestUnaryAuthInterceptor(t *testing.T) {
	verifier, privateKey := testVerifier(t)
	userID := uuid.New()
	validToken := signAccessToken(t, privateKey, userID, "customer")

	const (
		protectedMethod = "/bank.customer.v1.CustomerService/GetCustomer"
		publicMethod    = "/bank.customer.v1.CustomerService/Login"
	)

	publicMethods := map[string]bool{publicMethod: true}

	t.Run("без токена — Unauthenticated", func(t *testing.T) {
		interceptor := UnaryAuthInterceptor(verifier, publicMethods)
		_, err := interceptor(
			context.Background(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: protectedMethod},
			passThroughHandler,
		)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("ожидали Unauthenticated, получили %v", err)
		}
	})

	t.Run("битый токен — Unauthenticated", func(t *testing.T) {
		interceptor := UnaryAuthInterceptor(verifier, publicMethods)
		incomingContext := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs("authorization", "Bearer broken.token.value"),
		)
		_, err := interceptor(
			incomingContext,
			nil,
			&grpc.UnaryServerInfo{FullMethod: protectedMethod},
			passThroughHandler,
		)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("ожидали Unauthenticated, получили %v", err)
		}
	})

	t.Run("валидный токен — Claims в контексте", func(t *testing.T) {
		interceptor := UnaryAuthInterceptor(verifier, publicMethods)
		incomingContext := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs("authorization", "Bearer "+validToken),
		)
		capturing := &contextCapturingHandler{}
		_, err := interceptor(
			incomingContext,
			nil,
			&grpc.UnaryServerInfo{FullMethod: protectedMethod},
			capturing.Handle,
		)
		if err != nil {
			t.Fatalf("не ожидали ошибку: %v", err)
		}

		claims, err := authjwt.ClaimsFromContext(capturing.capturedContext)
		if err != nil {
			t.Fatalf("Claims не найдены в контексте: %v", err)
		}
		if claims.Subject != userID {
			t.Fatalf("ожидали subject %v, получили %v", userID, claims.Subject)
		}
		if claims.Role != "customer" {
			t.Fatalf("ожидали role customer, получили %q", claims.Role)
		}
	})

	t.Run("публичный метод без токена", func(t *testing.T) {
		interceptor := UnaryAuthInterceptor(verifier, publicMethods)
		_, err := interceptor(
			context.Background(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: publicMethod},
			passThroughHandler,
		)
		if err != nil {
			t.Fatalf("не ожидали ошибку: %v", err)
		}
	})
}