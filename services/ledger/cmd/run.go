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

	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
	grpcserver "github.com/Mazik-kun/mini-core/services/ledger/internal/adapters/inbound"
	accountgrpc "github.com/Mazik-kun/mini-core/services/ledger/internal/adapters/outbound/accountgrc"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/adapters/outbound/kafka"
	kafkaPkg "github.com/Mazik-kun/mini-core/pkg/kafka"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/adapters/outbound/postgres"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/config"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/app"
)

func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	// подключение к бд
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	defer pool.Close()

	
	queries := db.New(pool)
	ledgerRepo := postgres.NewLedgerRepo(queries)

	accountClient, err := accountgrpc.New(cfg.AccountGRPCAddr)

	if err != nil{
		return fmt.Errorf("account grpc: %w", err)
	}

	transferUC := app.NewTransferUseCase(pool, ledgerRepo, accountClient)
	depositUC := app.NewDepositUseCase(pool, ledgerRepo, accountClient)
	balanceUC := app.NewGetBalanceUseCase(ledgerRepo)
	listUC := app.NewListTransactionsUseCase(ledgerRepo)
	
	outboxWorker := kafka.NewOutboxWorker(queries, cfg.KafkaBrokers, cfg.KafkaTopic, log)
	// 4. gRPC Сервер
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
	
	ledgerServer := grpcserver.NewServer(transferUC, depositUC, balanceUC, listUC, log)
	ledgerv1.RegisterLedgerServiceServer(grpcServer, ledgerServer)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	// 5. Запуск
	g, gctx := errgroup.WithContext(ctx)
	if err := kafkaPkg.EnsureTopicExists(cfg.KafkaBrokers, cfg.KafkaTopic, 1, log); err != nil{
		return fmt.Errorf("ensure kafka topic: %w", err)
	}
	g.Go(func() error{
		return outboxWorker.Run(gctx)
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