package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	customerkafka "github.com/Mazik-kun/mini-core/services/customer/internal/adapters/kafka"
	grpcserver "github.com/Mazik-kun/mini-core/services/customer/internal/adapters/grpcserver"
	"github.com/Mazik-kun/mini-core/services/customer/internal/adapters/postgres"
	"github.com/Mazik-kun/mini-core/services/customer/internal/adapters/postgres/db"
	"github.com/Mazik-kun/mini-core/services/customer/internal/app"
	"github.com/Mazik-kun/mini-core/services/customer/internal/config"
)

// publicMethods — gRPC-методы, доступные без access-токена.
// В customer-сервисе публичных методов нет, только служебные.
var publicMethods = map[string]bool{
	"/grpc.health.v1.Health/Check":                                   true,
	"/grpc.health.v1.Health/Watch":                                   true,
	"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      true,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	publicKey, err := authjwt.LoadPublicKeyFromFile(cfg.JWTPublicKeyPath)
	if err != nil {
		log.Error("load jwt public key", "err", err)
		os.Exit(1)
	}
	jwtVerifier := authjwt.NewVerifier(publicKey)

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	dedupeStore := kafkautil.NewRedisDedupeStoreFromAddr(cfg.RedisAddr)
	defer dedupeStore.Close()

	queries := db.New(pool)
	repo := postgres.NewCustomerRepository(queries)

	getCustomer := app.NewGetCustomerUseCase(repo)
	getCustomerStatus := app.NewGetCustomerStatusUseCase(repo)
	listCustomers := app.NewListCustomersUseCase(repo)
	updateProfile := app.NewUpdateProfileUseCase(repo)
	createCustomer := app.NewCreateCustomerUseCase(repo)

	// Задел под KYC-консьюмер (kafka.events) — отдельная задача.
	_ = app.NewApplyKycDecisionUseCase(repo)

	srv := grpcserver.NewCustomerServer(
		getCustomer,
		getCustomerStatus,
		listCustomers,
		updateProfile,
		log,
	)

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Error("listen", "err", err, "addr", cfg.GRPCAddr)
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RequestID(),
			interceptors.Logging(log),
			interceptors.Recovery(log),
			interceptors.UnaryAuthInterceptor(jwtVerifier, publicMethods),
		),
	)
	customerv1.RegisterCustomerServiceServer(grpcSrv, srv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	reflection.Register(grpcSrv)

	customerConsumer := customerkafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaGroupID,
		dedupeStore,
		createCustomer,
		log,
	)

	go func() {
		log.Info("kafka consumer started",
			"topic", customerkafka.AuthEventsTopic, "group", cfg.KafkaGroupID)
		if err := customerConsumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("kafka consumer stopped", "err", err)
		}
	}()

	go func() {
		log.Info("grpc server started", "addr", cfg.GRPCAddr)
		if err := grpcSrv.Serve(listener); err != nil {
			log.Error("grpc serve", "err", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	grpcSrv.GracefulStop()
}