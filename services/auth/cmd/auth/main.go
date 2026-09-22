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
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	addr := os.Getenv("AUTH_GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "addr", addr, "err", err)
		os.Exit(1)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer()
	srv := newAuthServer(logger)
	authv1.RegisterAuthServiceServer(grpcServer, srv)

	reflection.Register(grpcServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
