package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	"github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/grpcserver"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/hasher"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/httpserver"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/jwt"
	authkafka "github.com/Mazik-kun/mini-core/services/auth/internal/adapters/kafka"
	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/postgres"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/postgres/db"
	"github.com/Mazik-kun/mini-core/services/auth/internal/app"
	"github.com/Mazik-kun/mini-core/services/auth/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 30 * 24 * time.Hour
)

// publicMethods — gRPC-методы, доступные без access-токена.
var publicMethods = map[string]bool{
	"/bank.auth.v1.AuthService/Register":      true,
	"/bank.auth.v1.AuthService/Login":         true,
	"/bank.auth.v1.AuthService/Refresh":       true,
	"/bank.auth.v1.AuthService/ValidateToken": true,

	// Служебные: иначе healthcheck и reflection потребуют токен.
	"/grpc.health.v1.Health/Check":                                   true,
	"/grpc.health.v1.Health/Watch":                                   true,
	"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      true,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	userRepo := postgres.NewUserRepository(queries)
	refreshRepo := postgres.NewRefreshTokenRepository(queries)

	argon := hasher.NewArgon2()

	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	dedupeStore := kafkautil.NewRedisDedupeStoreFromAddr(cfg.RedisAddr)
	defer dedupeStore.Close()
	if cfg.BootstrapAdminEmail != "" && cfg.BootstrapAdminPassword != "" {
		if err := app.BootstrapAdmin(
			ctx,
			userRepo,
			argon,
			cfg.BootstrapAdminEmail,
			cfg.BootstrapAdminPassword,
		); err != nil {
			logger.Error("bootstrap admin", "err", err)
			os.Exit(1)
		}
	}

	privateKeyPEM, err := os.ReadFile(cfg.JWTPrivateKeyPath)
	if err != nil {
		logger.Error("read jwt private key", "err", err)
		os.Exit(1)
	}
	publicKeyPEM, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		logger.Error("read jwt public key", "err", err)
		os.Exit(1)
	}

	issuer, err := jwt.NewIssuer(privateKeyPEM, accessTTL)
	if err != nil {
		logger.Error("jwt issuer", "err", err)
		os.Exit(1)
	}
	verifier, err := jwt.NewVerifier(publicKeyPEM)
	if err != nil {
		logger.Error("jwt verifier", "err", err)
		os.Exit(1)
	}
	publicKey, err := jwt.LoadPublicKey(publicKeyPEM)
	if err != nil {
		logger.Error("load jwt public key", "err", err)
		os.Exit(1)
	}
	authVerifier := authjwt.NewVerifier(publicKey)

	registerUC := app.NewRegisterUseCase(userRepo, argon, producer)
	loginUC := app.NewLoginUseCase(userRepo, argon, issuer, refreshRepo, refreshTTL)
	refreshUC := app.NewRefreshUseCase(refreshRepo, userRepo, issuer, refreshTTL)
	logoutUC := app.NewLogoutUseCase(refreshRepo)
	validateTokenUC := app.NewValidateTokenUseCase(verifier)
	createOfficerUC := app.NewCreateOfficerUseCase(userRepo, argon)

	applyCustomerCreatedUC := app.NewApplyCustomerCreatedUseCase(userRepo)

	customerEventsConsumer := authkafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaGroupID,
		dedupeStore,
		applyCustomerCreatedUC,
		logger,
	)
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Error("listen", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RequestID(),
			interceptors.Logging(logger),
			interceptors.Recovery(logger),
			interceptors.UnaryAuthInterceptor(authVerifier, publicMethods),
		),
	)

	srv := grpcserver.NewAuthServer(
		registerUC,
		loginUC,
		refreshUC,
		logoutUC,
		validateTokenUC,
		createOfficerUC,
		logger,
	)
	authv1.RegisterAuthServiceServer(grpcServer, srv)

	reflection.Register(grpcServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	httpSrv, err := httpserver.NewServer(cfg.HTTPAddr, publicKey, cfg.JWTKeyID, logger)
	if err != nil {
		logger.Error("http server", "err", err)
		os.Exit(1)
	}

	go runHTTPServer(ctx, httpSrv, logger)

	go func() {
		logger.Info("customer.events consumer started", "group", cfg.KafkaGroupID)
		if err := customerEventsConsumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("customer.events consumer stopped", "err", err)
		}
	}()
	go stopGRPCOnContextDone(ctx, grpcServer)

	logger.Info("auth service started",
		"grpc_addr", cfg.GRPCAddr,
		"http_addr", cfg.HTTPAddr,
		"kafka_brokers", cfg.KafkaBrokers,
	)
	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("serve grpc", "err", err)
		os.Exit(1)
	}
}

func runHTTPServer(ctx context.Context, srv *httpserver.Server, logger *slog.Logger) {
	if err := srv.Run(ctx); err != nil {
		logger.Error("http server", "err", err)
	}
}

func stopGRPCOnContextDone(ctx context.Context, grpcServer *grpc.Server) {
	<-ctx.Done()
	grpcServer.GracefulStop()
}