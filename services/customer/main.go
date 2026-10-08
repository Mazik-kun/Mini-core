package main

import (
	"context"
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
	grpcserver "github.com/Mazik-kun/mini-core/services/customer/internal/adapters/grpcserver"
	"github.com/Mazik-kun/mini-core/services/customer/internal/adapters/postgres"
	"github.com/Mazik-kun/mini-core/services/customer/internal/adapters/postgres/db"
	"github.com/Mazik-kun/mini-core/services/customer/internal/app"
	"github.com/Mazik-kun/mini-core/services/customer/internal/config"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	repo := postgres.NewCustomerRepository(queries)

	// Use cases, вызываемые gRPC-слоем.
	getCustomer := app.NewGetCustomerUseCase(repo)
	getCustomerStatus := app.NewGetCustomerStatusUseCase(repo)
	listCustomers := app.NewListCustomersUseCase(repo)
	updateProfile := app.NewUpdateProfileUseCase(repo)

	// Use cases для консьюмеров Kafka (auth.user_registered, kyc.events).
	// Подключение к Kafka — отдельная задача; здесь оставлены как задел.
	_ = app.NewCreateCustomerUseCase(repo)
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

	grpcSrv := grpc.NewServer()
	customerv1.RegisterCustomerServiceServer(grpcSrv, srv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	reflection.Register(grpcSrv)

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