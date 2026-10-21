// Package grpcconn создаёт исходящие gRPC-соединения с общим набором интерцепторов.
package grpcconn

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
)

type Options struct {
	Name    string // метка target в метриках: "auth" | "customer"
	Addr    string
	TLS     bool
	Timeout time.Duration
	Metrics *observability.Metrics
	Log     *slog.Logger
}

// Dial создаёт *grpc.ClientConn (lazy: реальное соединение — при первом вызове).
//
// Порядок (спека §7): otelgrpc (client span + traceparent) → forward (request-id,
// authorization, x-user-id из ctx) → timeout → observe (метрики + лог).
func Dial(o Options) (*grpc.ClientConn, error) {
	creds := insecure.NewCredentials()
	if o.TLS {
		creds = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	}
	conn, err := grpc.NewClient(o.Addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			forwardInterceptor(),
			timeoutInterceptor(o.Timeout),
			observeInterceptor(o.Name, o.Metrics, o.Log),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s (%s): %w", o.Name, o.Addr, err)
	}
	return conn, nil
}
