package httpserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/handlers"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type staticKeys struct{ pub ed25519.PublicKey }

func (s staticKeys) KeyByID(context.Context, string) (ed25519.PublicKey, error) { return s.pub, nil }

type allowAll struct{}

func (allowAll) Allow(context.Context, string) (bool, time.Duration, error) { return true, 0, nil }

// --- Фейковые клиенты для ручных хендлеров ---

type stubCustomers struct{}

func (stubCustomers) GetCustomer(_ context.Context, in *customerv1.GetCustomerRequest, _ ...grpc.CallOption) (*customerv1.Customer, error) {
	return &customerv1.Customer{CustomerId: in.CustomerId}, nil
}
func (stubCustomers) UpdateProfile(context.Context, *customerv1.UpdateProfileRequest, ...grpc.CallOption) (*customerv1.Customer, error) {
	return &customerv1.Customer{}, nil
}
func (stubCustomers) GetCustomerStatus(context.Context, *customerv1.GetCustomerStatusRequest, ...grpc.CallOption) (*customerv1.GetCustomerStatusResponse, error) {
	return &customerv1.GetCustomerStatusResponse{}, nil
}
func (stubCustomers) ListCustomers(_ context.Context, in *customerv1.ListCustomersRequest, _ ...grpc.CallOption) (*customerv1.ListCustomersResponse, error) {
	return &customerv1.ListCustomersResponse{}, nil
}

var _ port.CustomerClient = stubCustomers{}

type stubAuth struct{}

func (stubAuth) Register(context.Context, *authv1.RegisterRequest, ...grpc.CallOption) (*authv1.RegisterResponse, error) {
	return &authv1.RegisterResponse{}, nil
}
func (stubAuth) Login(context.Context, *authv1.LoginRequest, ...grpc.CallOption) (*authv1.LoginResponse, error) {
	return &authv1.LoginResponse{}, nil
}
func (stubAuth) Refresh(context.Context, *authv1.RefreshRequest, ...grpc.CallOption) (*authv1.LoginResponse, error) {
	return &authv1.LoginResponse{}, nil
}
func (stubAuth) Logout(context.Context, *authv1.LogoutRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (stubAuth) CreateOfficer(context.Context, *authv1.CreateOfficerRequest, ...grpc.CallOption) (*authv1.RegisterResponse, error) {
	return &authv1.RegisterResponse{}, nil
}

var _ port.AuthClient = stubAuth{}

// --- Тесты ---

func newTestRouter(t *testing.T) (http.Handler, string, string) {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	r := NewRouter(Deps{
		Log:         log,
		Metrics:     observability.NewMetrics(prometheus.NewRegistry()),
		Me:          handlers.NewMe(stubCustomers{}, log),
		Auth:        handlers.NewAuth(stubAuth{}),
		Customer:    handlers.NewCustomer(stubCustomers{}),
		Health:      handlers.NewHealth(),
		JWKS:        staticKeys{pub},
		UserLimiter: allowAll{},
		IPLimiter:   allowAll{},
	})

	customerID := uuid.NewString()
	mk := func(cid string) string {
		c := jwtlib.MapClaims{"sub": uuid.NewString(), "role": "CLIENT", "exp": time.Now().Add(time.Minute).Unix()}
		if cid != "" {
			c["customer_id"] = cid
		}
		tok := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, c)
		tok.Header["kid"] = "k1"
		s, _ := tok.SignedString(priv)
		return s
	}
	return r, mk(customerID), customerID
}

func do(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRouter_MeRoutesAreNotCapturedByGrpcGateway(t *testing.T) {
	h, tok, cid := newTestRouter(t)

	rec := do(h, http.MethodGet, "/v1/customers/me", tok, "")
	// Теперь ручной хендлер должен вернуть 200 и JSON с customer_id
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), cid) {
		t.Fatalf("/me must be served by manual handler: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, http.MethodGet, "/v1/customers/me/status", tok, "")
	if rec.Code != 200 {
		t.Fatalf("/me/status: %d %s", rec.Code, rec.Body)
	}
}

func TestRouter_OtherCustomerRoutesGoToManualHandlers(t *testing.T) {
	h, tok, _ := newTestRouter(t)
	
	// GET /v1/customers (List)
	rec := do(h, http.MethodGet, "/v1/customers", tok, "")
	if rec.Code != 200 {
		t.Errorf("/v1/customers: %d %s", rec.Code, rec.Body)
	}
	
	// GET /v1/customers/{id}
	rec = do(h, http.MethodGet, "/v1/customers/"+uuid.NewString(), tok, "")
	if rec.Code != 200 {
		t.Errorf("/v1/customers/{id}: %d %s", rec.Code, rec.Body)
	}
}

func TestRouter_ProtectedRoutesRequireJWT(t *testing.T) {
	h, _, _ := newTestRouter(t)
	for _, tc := range [][2]string{
		{http.MethodGet, "/v1/customers/me"},
		{http.MethodGet, "/v1/customers"},
		{http.MethodPost, "/v1/auth/officers"},
		{http.MethodGet, "/unknown"},
	} {
		if rec := do(h, tc[0], tc[1], "", ""); rec.Code != 401 {
			t.Errorf("%s %s: status = %d, want 401", tc[0], tc[1], rec.Code)
		}
	}
}

func TestRouter_PublicAuthRoutesAndHealth(t *testing.T) {
	h, _, _ := newTestRouter(t)
	
	// Передаем "{}" в теле, чтобы пройти parseProtoJSON
	rec := do(h, http.MethodPost, "/v1/auth/login", "", "{}")
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	
	if rec := do(h, http.MethodGet, "/healthz", "", ""); rec.Code != 200 {
		t.Fatalf("healthz: %d", rec.Code)
	}
	if rec := do(h, http.MethodGet, "/readyz", "", ""); rec.Code != 200 {
		t.Fatalf("readyz: %d", rec.Code)
	}
}

func TestRouter_ResponseCarriesRequestID(t *testing.T) {
	h, _, _ := newTestRouter(t)
	rec := do(h, http.MethodGet, "/v1/customers", "", "")
	// Middleware Auth вернет 401, но RequestID должен быть проставлен
	if rec.Header().Get("X-Request-ID") == "" || !strings.Contains(rec.Body.String(), rec.Header().Get("X-Request-ID")) {
		t.Fatalf("request_id missing: hdr=%q body=%s", rec.Header().Get("X-Request-ID"), rec.Body)
	}
}