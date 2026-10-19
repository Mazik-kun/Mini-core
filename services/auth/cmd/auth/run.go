package main

import (
	"context"
	"log/slog"
	"net"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/inbound/grpcserver"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/inbound/httpserver"
	authkafka "github.com/Mazik-kun/mini-core/services/auth/internal/adapters/outbound/kafka"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/outbound/hasher"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/outbound/jwt"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/outbound/postgres"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/auth/internal/config"
	"github.com/Mazik-kun/mini-core/services/auth/internal/core/app"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	// 1. Инфраструктура
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil { return err }
	defer pool.Close()

	producer := kafkautil.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	dedupeStore := kafkautil.NewRedisDedupeStoreFromAddr(cfg.RedisAddr)
	defer dedupeStore.Close()

	// 2. Репозитории и адаптеры
	queries := db.New(pool)
	userRepo := postgres.NewUserRepository(queries)
	refreshRepo := postgres.NewRefreshTokenRepository(queries)
	argon := hasher.NewArgon2()

	// 3. JWT
	issuer, err := jwt.NewIssuer(cfg.JWTPrivateKeyPath, cfg.JWTAccessTTL)
	if err != nil { return err }
	verifier, err := jwt.NewVerifier(cfg.JWTPublicKeyPath)
	if err != nil { return err }
	publicKey, err := jwt.LoadPublicKeyFromFile(cfg.JWTPublicKeyPath)
	if err != nil { return err }
	authVerifier := authjwt.NewVerifier(publicKey)

	// 4. Bootstrap
	if cfg.BootstrapAdminEmail != "" && cfg.BootstrapAdminPassword != "" {
		if err := app.BootstrapAdmin(ctx, userRepo, argon, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword); err != nil {
			return err
		}
	}

	// 5. Use Cases
	registerUC := app.NewRegisterUseCase(userRepo, argon, producer, cfg.AuthEventsTopic)
	loginUC := app.NewLoginUseCase(userRepo, argon, issuer, refreshRepo, cfg.JWTRefreshTTL)
	refreshUC := app.NewRefreshUseCase(refreshRepo, userRepo, issuer, cfg.JWTRefreshTTL)
	logoutUC := app.NewLogoutUseCase(refreshRepo)
	validateTokenUC := app.NewValidateTokenUseCase(verifier)
	createOfficerUC := app.NewCreateOfficerUseCase(userRepo, argon)
	applyCustomerCreatedUC := app.NewApplyCustomerCreatedUseCase(userRepo)

	// 6. Kafka Consumer
	customerEventsConsumer := authkafka.NewConsumer(
		cfg.KafkaBrokers, cfg.KafkaGroupID, dedupeStore, applyCustomerCreatedUC, logger,
	)

	// 7. gRPC Server
	grpcSrv := setupGRPC(cfg, logger, authVerifier, registerUC, loginUC, refreshUC, logoutUC, validateTokenUC, createOfficerUC)

	// 8. HTTP Server
	httpSrv, err := httpserver.NewServer(cfg.HTTPAddr, publicKey, cfg.JWTKeyID, logger)
	if err != nil { return err }

	// 9. Запуск
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil { return err }

	go func() {
		logger.Info("customer.events consumer started", "group", cfg.KafkaGroupID)
		if err := customerEventsConsumer.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("consumer error", "err", err)
		}
	}()

	go func() {
		if err := httpSrv.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("http server error", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		logger.Info("shutting down grpc server")
		grpcSrv.GracefulStop()
	}()

	logger.Info("auth service started", "grpc", cfg.GRPCAddr, "http", cfg.HTTPAddr)
	return grpcSrv.Serve(lis)
}

func setupGRPC(
	cfg *config.Config, logger *slog.Logger, authVerifier *authjwt.Verifier,
	register *app.RegisterUseCase, login *app.LoginUseCase, refresh *app.RefreshUseCase, logout *app.LogoutUseCase,
	validate *app.ValidateTokenUseCase, createOfficer *app.CreateOfficerUseCase,
) *grpc.Server {
	publicMethods := map[string]bool{
		"/bank.auth.v1.AuthService/Register":      true,
		"/bank.auth.v1.AuthService/Login":         true,
		"/bank.auth.v1.AuthService/Refresh":       true,
		"/bank.auth.v1.AuthService/ValidateToken": true,
		"/grpc.health.v1.Health/Check":            true,
		"/grpc.health.v1.Health/Watch":            true,
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RequestID(),
			interceptors.Logging(logger),
			interceptors.Recovery(logger),
			interceptors.UnaryAuthInterceptor(authVerifier, publicMethods),
		),
	)

	srv := grpcserver.NewAuthServer(register, login, refresh, logout, validate, createOfficer, logger)
	authv1.RegisterAuthServiceServer(grpcServer, srv)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	return grpcServer
}