package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/config"
	httpserver "github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/handlers"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/authgrpc"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/customergrpc"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/grpcconn"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/jwks"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/ratelimit"
)

// run — DI + жизненный цикл. Остановка: ctx отменён → HTTP Shutdown → закрытие gRPC conns
// → redis.Close() → flush трейсов. Ошибки закрытия логируются и не блокируют остальное.
func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	shutdownTracing, err := observability.SetupTracing(ctx, cfg.OTELEndpoint, "api-gateway")
	if err != nil {
		return fmt.Errorf("tracing: %w", err)
	}

	metrics := observability.NewMetrics(prometheus.NewRegistry())

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  time.Second,
		ReadTimeout:  300 * time.Millisecond, // rate limiter на горячем пути: быстро падаем → fail-closed
		WriteTimeout: 300 * time.Millisecond,
		MaxRetries:   1,
	})

	dial := grpcconn.Options{TLS: cfg.GRPCTLS, Timeout: cfg.GRPCTimeout, Metrics: metrics, Log: log}

	authOpts := dial
	authOpts.Addr = cfg.AuthGRPCAddr
	authClient, err := authgrpc.New(authOpts)
	if err != nil {
		return err
	}
	customerOpts := dial
	customerOpts.Addr = cfg.CustomerGRPCAddr
	customerClient, err := customergrpc.New(customerOpts)
	if err != nil {
		return err
	}

	defer func() {
		for name, closeFn := range map[string]func() error{
			"auth grpc": authClient.Close, "customer grpc": customerClient.Close, "redis": rdb.Close,
		} {
			if err := closeFn(); err != nil {
				log.Error("close failed", "resource", name, "error", err)
			}
		}
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(sctx); err != nil {
			log.Error("tracing shutdown failed", "error", err)
		}
	}()

	jwksCache := jwks.NewCache(
		jwks.NewHTTPFetcher(cfg.AuthJWKSURL, 5*time.Second),
		jwks.Options{
			TTL:         cfg.JWKSCacheTTL,
			MinInterval: cfg.JWKSRefreshMinInterval,
			Log:         log,
			OnRefresh: func(ok bool) {
				result := "ok"
				if !ok {
					result = "err"
				}
				metrics.JWKSRefresh.WithLabelValues(result).Inc()
			},
		})

	redisErr := func(op string) { metrics.RLRedisErrors.WithLabelValues(op).Inc() }
	userLimiter := ratelimit.New(rdb, cfg.RLUserBurst, cfg.RLUserRPS, redisErr)
	ipLimiter := ratelimit.New(rdb, cfg.RLIPBurst, cfg.RLIPRPS, redisErr)


	health := handlers.NewHealth(
		handlers.Check{Name: "auth", Fn: grpcHealth(authClient.Conn())},
		handlers.Check{Name: "customer", Fn: grpcHealth(customerClient.Conn())},
		handlers.Check{Name: "redis", Fn: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }},
	)

	router := httpserver.NewRouter(httpserver.Deps{
		Log:         log,
		Metrics:     metrics,
		Me:          handlers.NewMe(customerClient, log),
		Auth:        handlers.NewAuth(authClient),       // <--- ДОБАВЛЕНО
		Customer:    handlers.NewCustomer(customerClient), // <--- ДОБАВЛЕНО
		Health:      health,
		JWKS:        jwksCache,
		UserLimiter: userLimiter,
		IPLimiter:   ipLimiter,
		JWTIssuer:   cfg.JWTIssuer,
		CORSOrigins: cfg.CORSOrigins,
		TrustXFF:    cfg.TrustForwardedFor,
	})
	httpSrv := httpserver.New(cfg.HTTPAddr, router, cfg.ShutdownTimeout)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))
	metricsSrv := httpserver.New(cfg.MetricsAddr, metricsMux, cfg.ShutdownTimeout)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { jwksCache.Run(gctx); return nil })
	g.Go(func() error {
		log.Info("http server starting", "addr", cfg.HTTPAddr)
		return httpSrv.Run(gctx)
	})
	g.Go(func() error {
		log.Info("metrics server starting", "addr", cfg.MetricsAddr)
		return metricsSrv.Run(gctx)
	})
	return g.Wait()
}

func grpcHealth(conn *grpc.ClientConn) func(context.Context) error {
	client := healthpb.NewHealthClient(conn)
	return func(ctx context.Context) error {
		resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
		if err != nil {
			return err
		}
		if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
			return errors.New("not serving: " + resp.GetStatus().String())
		}
		return nil
	}
}
