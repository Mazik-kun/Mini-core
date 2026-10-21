package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"google.golang.org/grpc/codes"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
)

// Recovery превращает panic в 500 в едином формате и логирует стек.
func Recovery(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				log.ErrorContext(r.Context(), "panic recovered",
					"panic", rec, "stack", string(debug.Stack()),
					"method", r.Method, "path", r.URL.Path)
				apierr.WriteStatus(w, r, codes.Internal, "internal error")
			}()
			next.ServeHTTP(w, r)
		})
	}
}
