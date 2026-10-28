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

	documentv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/document/v1"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	grpcserver "github.com/Mazik-kun/mini-core/services/document/internal/adapters/inbound/grpcserver"
	minioadapter "github.com/Mazik-kun/mini-core/services/document/internal/adapters/outbound/minio"
	"github.com/Mazik-kun/mini-core/services/document/internal/adapters/outbound/postgres"
	"github.com/Mazik-kun/mini-core/services/document/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/document/internal/config"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/app"
)

var publicMethods = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/Watch": true,
}

// bucketInitTimeout — лимит на создание бакета при старте (fail-fast).
const bucketInitTimeout = 30 * time.Second

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

	storage, err := minioadapter.New(minioadapter.Options{
		Endpoint:       cfg.MinioEndpoint,
		PublicEndpoint: cfg.MinioPublicEndpoint,
		AccessKey:      cfg.MinioAccessKey,
		SecretKey:      cfg.MinioSecretKey,
		UseSSL:         cfg.MinioUseSSL,
		Bucket:         cfg.MinioBucket,
	})
	if err != nil {
		return err
	}
	bucketCtx, cancel := context.WithTimeout(ctx, bucketInitTimeout)
	err = storage.EnsureBucket(bucketCtx)
	cancel()
	if err != nil {
		return err
	}
	log.Info("minio bucket ready", "bucket", cfg.MinioBucket)

	queries := db.New(pool)
	repo := postgres.NewDocumentRepository(queries)

	initUpload := app.NewInitUploadUseCase(repo, storage, cfg.UploadURLTTL)
	confirmUpload := app.NewConfirmUploadUseCase(repo, storage, cfg.MaxSize)
	getDownloadURL := app.NewGetDownloadURLUseCase(repo, storage, cfg.DownloadURLTTL)
	listDocuments := app.NewListDocumentsUseCase(repo, cfg.DefaultPageSize, cfg.MaxPageSize)
	hasRequired := app.NewHasRequiredDocumentsUseCase(repo, cfg.RequiredDocTypes)

	srv := grpcserver.NewDocumentServer(initUpload, confirmUpload, getDownloadURL, listDocuments, hasRequired, log)

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
	documentv1.RegisterDocumentServiceServer(grpcSrv, srv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)

	go serveGRPC(log, grpcSrv, listener, cfg.GRPCAddr)

	<-ctx.Done()
	log.Info("shutting down")
	grpcSrv.GracefulStop()

	return nil
}

func serveGRPC(log *slog.Logger, srv *grpc.Server, listener net.Listener, addr string) {
	log.Info("grpc server started", "addr", addr)
	if err := srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Error("grpc serve", "err", err)
	}
}
