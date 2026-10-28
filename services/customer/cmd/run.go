package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	kafkautil "github.com/Mazik-kun/mini-core/pkg/kafka"
	grpcserver "github.com/Mazik-kun/mini-core/services/customer/internal/adapters/inbound/grpcserver"
	customerkafka "github.com/Mazik-kun/mini-core/services/customer/internal/adapters/outbound/kafka"
	"github.com/Mazik-kun/mini-core/services/customer/internal/adapters/outbound/postgres"
	"github.com/Mazik-kun/mini-core/services/customer/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/customer/internal/config"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/app"
)

// publicMethods — gRPC-методы, доступные без access-токена.
// Reflection больше не регистрируем, поэтому его методов здесь нет.
var publicMethods = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/Watch": true,
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	publicKey, err := authjwt.LoadPublicKeyFromFile(cfg.JWTPublicKeyPath)
	if err != nil {
		return err
	}
	jwtVerifier := authjwt.NewVerifier(publicKey)

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	dedupeStore := kafkautil.NewRedisDedupeStoreFromAddr(cfg.RedisAddr)
	defer dedupeStore.Close()

	queries := db.New(pool)
	repo := postgres.NewCustomerRepository(queries)

	getCustomer := app.NewGetCustomerUseCase(repo)
	getCustomerStatus := app.NewGetCustomerStatusUseCase(repo)
	listCustomers := app.NewListCustomersUseCase(repo, cfg.DefaultPageSize, cfg.MaxPageSize)
	updateProfile := app.NewUpdateProfileUseCase(repo)

	producer := kafkautil.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	createCustomer := app.NewCreateCustomerUseCase(
		repo,
		producer,
		cfg.KafkaCustomerEventsTopic,
	)

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
		return err
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

	customerConsumer := customerkafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaGroupID,
		cfg.KafkaAuthEventsTopic,
		dedupeStore,
		createCustomer,
		log,
	)

	applyKycDecision := app.NewApplyKycDecisionUseCase(repo)
	kycGroupID := cfg.KafkaGroupID + "-kyc"
	kycTopic := "kyc.events"
	
	kycConsumer := customerkafka.NewKycConsumer(
		cfg.KafkaBrokers,
		kycGroupID,
		kycTopic,
		dedupeStore,
		applyKycDecision,
		log,
	)
	
	go runKafkaConsumer(ctx, log, kycConsumer, kycTopic, kycGroupID)
	go runKafkaConsumer(ctx, log, customerConsumer, cfg.KafkaAuthEventsTopic, cfg.KafkaGroupID)
	go serveGRPC(log, grpcSrv, listener, cfg.GRPCAddr)

	<-ctx.Done()
	log.Info("shutting down")
	grpcSrv.GracefulStop()

	return nil
}

type KafkaRunnable interface {
	Run(ctx context.Context) error
}

func runKafkaConsumer(
	ctx context.Context,
	log *slog.Logger,
	consumer KafkaRunnable, 
	topic string,
	groupID string,
) {
	log.Info("kafka consumer started", "topic", topic, "group", groupID)
	if err := consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("kafka consumer stopped", "err", err)
	}
}

func serveGRPC(
	log *slog.Logger,
	srv *grpc.Server,
	listener net.Listener,
	addr string,
) {
	log.Info("grpc server started", "addr", addr)
	if err := srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Error("grpc serve", "err", err)
	}
}
