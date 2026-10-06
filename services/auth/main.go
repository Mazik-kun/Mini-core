package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/grpcserver"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/hasher"
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

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	userRepo := postgres.NewUserRepository(queries)
	argon := hasher.NewArgon2()

	registerUC := app.NewRegisterUseCase(userRepo, argon)
	loginUC := app.NewLoginUseCase(userRepo, argon)

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
		),
	)

	srv := grpcserver.NewAuthServer(registerUC, loginUC, logger)
	authv1.RegisterAuthServiceServer(grpcServer, srv)

	reflection.Register(grpcServer)
	
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info("auth service started", "addr", cfg.GRPCAddr)
	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("serve", "err", err)
		os.Exit(1)
	}
}