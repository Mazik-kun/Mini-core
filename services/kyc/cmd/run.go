package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	kycv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/kyc/v1"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	grpcserver "github.com/Mazik-kun/mini-core/services/kyc/internal/adapters/inbound/grpcserver"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/adapters/outbound/customergrpc"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/adapters/outbound/documentgrpc"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/adapters/outbound/kafka"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/adapters/outbound/postgres"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/config"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/app"
)

var publicMethods = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/Watch": true,
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil { return err }

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// JWT
	pubKey, err := authjwt.LoadPublicKeyFromFile(cfg.JWTPublicKeyPath)
	if err != nil { return err }
	jwtVerifier := authjwt.NewVerifier(pubKey)

	// Postgres
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil { return err }
	defer pool.Close()
	repo := postgres.NewKycRepository(pool)

	// Kafka
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	// gRPC Clients (Customer & Document)
	custCli, err := customergrpc.NewClient(cfg.CustomerGRPCAddr)
	if err != nil { return err }

	docCli, err := documentgrpc.NewClient(cfg.DocumentGRPCAddr)
	if err != nil { return err }

	// Use Cases
	submitUC := app.NewSubmitApplicationUseCase(repo, custCli, docCli, producer, cfg.KafkaKycTopic, []string{"PASSPORT", "SELFIE"})
	officerUC := app.NewOfficerActionsUseCase(repo, producer, cfg.KafkaKycTopic)
	getUC := app.NewGetApplicationUseCase(repo)
	listUC := app.NewListApplicationsUseCase(repo)

	// Передаем submitUC вместо nil
	srv := grpcserver.NewServer(submitUC, officerUC, getUC, listUC)

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil { return err }

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RequestID(),
			interceptors.Logging(log),
			interceptors.Recovery(log),
			interceptors.UnaryAuthInterceptor(jwtVerifier, publicMethods),
		),
	)
	kycv1.RegisterKycServiceServer(grpcSrv, srv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)

	// Запускаем воркер
	worker := app.NewAutoCheckWorker(repo, repo, producer, cfg.KafkaKycTopic, cfg.WorkerInterval, cfg.WorkerBatch, cfg.MaxAttempts, cfg.MinAge, log)
	go worker.Run(ctx)

	go func() {
		log.Info("grpc server started", "addr", cfg.GRPCAddr)
		if err := grpcSrv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Error("grpc serve", "err", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	
	done := make(chan struct{})
	go func() { grpcSrv.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		grpcSrv.Stop()
	}

	return nil
}