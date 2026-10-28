// Package config читает конфигурацию gateway из окружения (GATEWAY_*).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

const MaxBodyBytes = 1 << 20 // 1 MiB

type Config struct {
	HTTPAddr        string
	MetricsAddr     string
	ShutdownTimeout time.Duration

	AuthGRPCAddr     string
	CustomerGRPCAddr string
	DocumentGRPCAddr string
	KYCGRPCAddr      string
	GRPCTLS          bool
	GRPCTimeout      time.Duration

	AuthJWKSURL            string
	JWKSCacheTTL           time.Duration
	JWKSRefreshMinInterval time.Duration
	JWTIssuer              string // пусто = iss не проверяется

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	RLUserRPS   float64
	RLUserBurst int
	RLIPRPS     float64
	RLIPBurst   int

	CORSOrigins       []string
	TrustForwardedFor bool

	LogLevel     string
	OTELEndpoint string // пусто = трейсинг выключен
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	var errs []error
	dur := func(name string, def time.Duration) time.Duration {
		v, err := envDuration(name, def)
		if err != nil {
			errs = append(errs, err)
		}
		return v
	}
	integer := func(name string, def int) int {
		v, err := envInt(name, def)
		if err != nil {
			errs = append(errs, err)
		}
		return v
	}
	float := func(name string, def float64) float64 {
		v, err := envFloat(name, def)
		if err != nil {
			errs = append(errs, err)
		}
		return v
	}
	boolean := func(name string, def bool) bool {
		v, err := envBool(name, def)
		if err != nil {
			errs = append(errs, err)
		}
		return v
	}

	cfg := &Config{
		HTTPAddr:        getEnv("GATEWAY_HTTP_ADDR", ":8080"),
		MetricsAddr:     getEnv("GATEWAY_METRICS_ADDR", ":9090"),
		ShutdownTimeout: dur("GATEWAY_SHUTDOWN_TIMEOUT", 15*time.Second),

		AuthGRPCAddr: getEnv("GATEWAY_AUTH_GRPC_ADDR", "auth:50051"),
		// customer-service по умолчанию слушает :50052 (см. его config.go),
		// поэтому дефолт здесь customer:50052, а не 50051 как в спеке.
		CustomerGRPCAddr: getEnv("GATEWAY_CUSTOMER_GRPC_ADDR", "customer:50052"),
		DocumentGRPCAddr: getEnv("GATEWAY_DOCUMENT_GRPC_ADDR", "document:50054"),
		KYCGRPCAddr:      getEnv("GATEWAY_KYC_GRPC_ADDR", "kyc:50055"),
		GRPCTLS:          boolean("GATEWAY_GRPC_TLS", false),
		GRPCTimeout:      dur("GATEWAY_GRPC_TIMEOUT", 5*time.Second),

		AuthJWKSURL:            getEnv("GATEWAY_AUTH_JWKS_URL", "http://auth:8081/.well-known/jwks.json"),
		JWKSCacheTTL:           dur("GATEWAY_JWKS_CACHE_TTL", 15*time.Minute),
		JWKSRefreshMinInterval: dur("GATEWAY_JWKS_REFRESH_MIN_INTERVAL", time.Minute),
		JWTIssuer:              getEnv("GATEWAY_JWT_ISSUER", ""),

		RedisAddr:     getEnv("GATEWAY_REDIS_ADDR", ""),
		RedisPassword: os.Getenv("GATEWAY_REDIS_PASSWORD"),
		RedisDB:       integer("GATEWAY_REDIS_DB", 0),

		RLUserRPS:   float("GATEWAY_RL_USER_RPS", 10),
		RLUserBurst: integer("GATEWAY_RL_USER_BURST", 20),
		RLIPRPS:     float("GATEWAY_RL_IP_RPS", 2),
		RLIPBurst:   integer("GATEWAY_RL_IP_BURST", 5),

		CORSOrigins:       splitCSV(os.Getenv("GATEWAY_CORS_ORIGINS")),
		TrustForwardedFor: boolean("GATEWAY_TRUST_FORWARDED_FOR", false),

		LogLevel:     getEnv("GATEWAY_LOG_LEVEL", "info"),
		OTELEndpoint: otelEndpoint(),
	}

	if cfg.RedisAddr == "" {
		errs = append(errs, errors.New("GATEWAY_REDIS_ADDR is not set"))
	}
	if cfg.RLUserRPS <= 0 || cfg.RLUserBurst <= 0 {
		errs = append(errs, errors.New("GATEWAY_RL_USER_RPS and GATEWAY_RL_USER_BURST must be > 0"))
	}
	if cfg.RLIPRPS <= 0 || cfg.RLIPBurst <= 0 {
		errs = append(errs, errors.New("GATEWAY_RL_IP_RPS and GATEWAY_RL_IP_BURST must be > 0"))
	}
	if cfg.GRPCTimeout <= 0 {
		errs = append(errs, errors.New("GATEWAY_GRPC_TIMEOUT must be > 0"))
	}
	if cfg.JWKSCacheTTL <= 0 {
		errs = append(errs, errors.New("GATEWAY_JWKS_CACHE_TTL must be > 0"))
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid gateway config: %w", errors.Join(errs...))
	}
	return cfg, nil
}

// otelEndpoint: не задана — дефолт из спеки (jaeger:4317); задана пустой/"off" — трейсинг выключен.
func otelEndpoint() string {
	v, ok := os.LookupEnv("GATEWAY_OTEL_ENDPOINT")
	if !ok {
		return "jaeger:4317"
	}
	v = strings.TrimSpace(v)
	if v == "off" {
		return ""
	}
	return v
}

func getEnv(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func envDuration(name string, def time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func envInt(name string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func envFloat(name string, def float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func envBool(name string, def bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func splitCSV(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
