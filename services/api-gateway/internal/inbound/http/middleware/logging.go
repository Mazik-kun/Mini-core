package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

// Logging пишет access-лог после ответа. user_id и grpc_code берутся из *reqctx.Info,
// который заполняют auth-middleware и apierr.
func Logging(log *slog.Logger, trustXFF bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := wrapWriter(w)
			next.ServeHTTP(rec, r)

			attrs := []any{
				"request_id", reqctx.RequestID(r.Context()),
				"method", r.Method,
				"route", RouteLabel(r),
				"status", rec.status,
				"latency_ms", float64(time.Since(start).Microseconds()) / 1000,
				"remote_ip", ClientIP(r, trustXFF),
			}
			if info := reqctx.InfoFrom(r.Context()); info != nil {
				if info.UserID != "" {
					attrs = append(attrs, "user_id", info.UserID)
				}
				if info.GRPCCode != "" {
					attrs = append(attrs, "grpc_code", info.GRPCCode)
				}
			}
			level := slog.LevelInfo
			if rec.status >= 500 {
				level = slog.LevelError
			}
			log.Log(r.Context(), level, "http request", attrs...)
		})
	}
}
