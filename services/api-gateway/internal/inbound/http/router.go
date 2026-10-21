package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/config"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/handlers"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/middleware"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type Deps struct {
	Log     *slog.Logger
	Metrics *observability.Metrics

	Gateway http.Handler // grpc-gateway (сгенерированные хендлеры)
	Me      *handlers.Me
	Health  *handlers.Health
	Auth     *handlers.Auth
	Customer *handlers.Customer

	JWKS        port.JWKSProvider
	UserLimiter port.RateLimiter
	IPLimiter   port.RateLimiter

	JWTIssuer   string
	CORSOrigins []string
	TrustXFF    bool
}


func NewRouter(d Deps) http.Handler {
	inner := http.NewServeMux()
	
	inner.HandleFunc("GET /v1/customers/me", d.Me.GetMe)
	inner.HandleFunc("PATCH /v1/customers/me", d.Me.UpdateMe)
	inner.HandleFunc("GET /v1/customers/me/status", d.Me.GetStatus)
	
	inner.HandleFunc("POST /v1/auth/register", d.Auth.Register)
	inner.HandleFunc("POST /v1/auth/login", d.Auth.Login)
	inner.HandleFunc("POST /v1/auth/refresh", d.Auth.Refresh)
	inner.HandleFunc("POST /v1/auth/logout", d.Auth.Logout)
	inner.HandleFunc("POST /v1/auth/officers", d.Auth.CreateOfficer)
	
	inner.HandleFunc("GET /v1/customers", d.Customer.List)
	inner.HandleFunc("GET /v1/customers/{customer_id}", d.Customer.GetByID) 



	chain := middleware.Chain(
		middleware.Recovery(d.Log),
		middleware.RequestID(),
		middleware.Tracing(),
		middleware.Logging(d.Log, d.TrustXFF),
		middleware.Metrics(d.Metrics),
		middleware.CORS(d.CORSOrigins),
		middleware.BodyLimit(config.MaxBodyBytes),
		middleware.Auth(d.JWKS, d.JWTIssuer, d.Log),
		middleware.RateLimit(d.UserLimiter, d.IPLimiter, d.TrustXFF, d.Metrics, d.Log),
	)

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", d.Health.Live)
	root.HandleFunc("GET /readyz", d.Health.Ready)
	root.Handle("/", chain(inner))
	return root
}
