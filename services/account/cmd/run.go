package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	accountv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/account/v1"
	"github.com/Mazik-kun/mini-core/services/account/internal/adapters/inbound/grpcserver"
	"github.com/Mazik-kun/mini-core/services/account/internal/adapters/outbound/kafka"
	"github.com/Mazik-kun/mini-core/services/account/internal/adapters/outbound/postgres"
	"github.com/Mazik-kun/mini-core/services/account/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/account/internal/config"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/app"
)

func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	// 1. Подключение к БД
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	accountRepo := postgres.NewAccountRepo(queries)

	// 2. ВРЕМЕННАЯ ЗАГЛУШКА вместо gRPC клиента
	customerCli := &app.MockCustomerClient{}

	// Use Cases
	openUC := app.NewOpenAccountUseCase(accountRepo, customerCli)
	getUC := app.NewGetAccountUseCase(accountRepo)
	listUC := app.NewListAccountsUseCase(accountRepo)
	blockUC := app.NewBlockAccountUseCase(accountRepo)     // <-- ДОБАВИТЬ
	closeUC := app.NewCloseAccountUseCase(accountRepo)     // <-- ДОБАВИТЬ
	balanceUC := app.NewGetBalanceUseCase(accountRepo)     // <-- ДОБАВИТЬ

	// gRPC Сервер
		grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
				defer func() {
					if r := recover(); r != nil {
						log.Error("panic in gRPC handler", "method", info.FullMethod, "panic", r)
						err = status.Error(codes.Internal, "internal server error")
					}
				}()
				return handler(ctx, req)
			},
		),
	)
	accServer := grpcserver.NewServer(openUC, getUC, listUC, blockUC, closeUC, balanceUC, log) // <-- Передать в сервер
	accountv1.RegisterAccountServiceServer(grpcServer, accServer)

	// Health check
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	kafkaConsumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, accountRepo, openUC, listUC, log)
	defer kafkaConsumer.Close()

	// 5. Запуск сервера
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func()error{
		log.Info("starting kafka consumer")
		return kafkaConsumer.Run(gctx)
	})
	
	g.Go(func() error {
		lis, err := net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			return fmt.Errorf("tcp listen: %w", err)
		}
		log.Info("gRPC server started", "addr", cfg.GRPCAddr)
		
		go func() {
			<-gctx.Done()
			log.Info("shutting down gRPC server gracefully...")
			healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
			
			stopped := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				close(stopped)
			}()
			select {
			case <-stopped:
			case <-time.After(5 * time.Second):
				grpcServer.Stop()
			}
		}()

		return grpcServer.Serve(lis)
	})

	return g.Wait()
}