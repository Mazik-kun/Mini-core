package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/pkg/config"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/grpc/interceptors"
	"github.com/Mazik-kun/mini-core/services/auth/internal/adapters/postgres/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	config.LoadEnv()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := os.Getenv("AUTH_GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL IS NOT SET")
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "addr", addr, "err", err)
		os.Exit(1)
	}
	defer lis.Close()

	poolCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(poolCtx, dbURL)
	if err != nil {
		logger.Error("failed to create pool", "err", err)
		os.Exit(1)
	}

	if err := pool.Ping(poolCtx); err != nil {
		logger.Error("failed to ping database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	queries := db.New(pool)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.Recovery(logger)),
	)
	srv := newAuthServer(logger, queries)
	authv1.RegisterAuthServiceServer(grpcServer, srv)

	reflection.Register(grpcServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc serve failed", "err", err)
		}
	}()
	logger.Info("grpc server listening", "addr", addr)

	<-ctx.Done()

	logger.Info("shutting down")

	done := make(chan struct{})

	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		logger.Info("stopped cleanly")
	case <-time.After(10 * time.Second):
		logger.Warn("forced stop after timeout")
		grpcServer.Stop()
	}

}
