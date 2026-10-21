package grpcconn

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

// forwardInterceptor проставляет в исходящие metadata:
//   - authorization: Bearer <raw JWT> — ОБЯЗАТЕЛЬНО, downstream сам валидирует токен;
//   - x-request-id, x-user-id — advisory (только для логов, не для авторизации).
//
// Ключи, уже выставленные grpc-gateway (WithMetadata / Authorization), не дублируются.
func forwardInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		var add []string
		if len(md.Get("x-request-id")) == 0 {
			if id := reqctx.RequestID(ctx); id != "" {
				add = append(add, "x-request-id", id)
			}
		}
		if len(md.Get("authorization")) == 0 {
			if tok := reqctx.Token(ctx); tok != "" {
				add = append(add, "authorization", "Bearer "+tok)
			}
		}
		if len(md.Get("x-user-id")) == 0 {
			if c, ok := reqctx.Claims(ctx); ok {
				add = append(add, "x-user-id", c.Subject.String())
			}
		}
		if len(add) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, add...)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// timeoutInterceptor ограничивает каждый вызов (если у родительского ctx дедлайн короче — он главнее).
func timeoutInterceptor(d time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// observeInterceptor — client-side метрики и лог ошибок downstream.
func observeInterceptor(target string, m *observability.Metrics, log *slog.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		code := status.Code(err)

		if m != nil {
			m.GRPCClientDuration.WithLabelValues(target, method, code.String()).Observe(time.Since(start).Seconds())
		}
		if err != nil && log != nil {
			level := slog.LevelDebug // ожидаемые бизнес-ошибки (NOT_FOUND, ...) — не шум
			switch code {
			case codes.Internal, codes.Unknown, codes.Unavailable, codes.DeadlineExceeded, codes.DataLoss:
				level = slog.LevelError
			}
			log.Log(ctx, level, "grpc call failed",
				"target", target, "method", method, "grpc_code", code.String(),
				"request_id", reqctx.RequestID(ctx), "error", err)
		}
		return err
	}
}
