package middleware

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

// RateLimit — fail-closed. Есть claims (прошли auth) → лимит по user (rl:user:{sub}),
// иначе (публичные эндпоинты) → по IP (rl:ip:{ip}). Любая ошибка лимитера → 503.
func RateLimit(user, ip port.RateLimiter, trustXFF bool, m *observability.Metrics, log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				limiter port.RateLimiter
				key     string
				scope   string
			)
			if c, ok := reqctx.Claims(r.Context()); ok {
				limiter, key, scope = user, "rl:user:"+c.Subject.String(), "user"
			} else {
				limiter, key, scope = ip, "rl:ip:"+ClientIP(r, trustXFF), "ip"
			}

			allowed, retry, err := limiter.Allow(r.Context(), key)
			if err != nil {
				log.ErrorContext(r.Context(), "rate limiter failed (fail-closed)", "scope", scope, "error", err)
				w.Header().Set("Retry-After", "1")
				apierr.WriteStatus(w, r, codes.Unavailable, "rate limiter unavailable")
				return
			}
			if !allowed {
				m.RLRejected.WithLabelValues(scope).Inc()
				secs := int(math.Ceil(retry.Seconds()))
				if secs < 1 {
					secs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				apierr.WriteStatus(w, r, codes.ResourceExhausted, "rate limit exceeded",
					&errdetails.RetryInfo{RetryDelay: durationpb.New(roundMs(retry))})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func roundMs(d time.Duration) time.Duration { return d.Round(time.Millisecond) }
