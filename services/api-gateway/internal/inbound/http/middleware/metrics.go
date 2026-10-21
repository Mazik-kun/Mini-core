package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/observability"
)

// Metrics — RED-метрики. Метка route — шаблон, а не сырой путь.
func Metrics(m *observability.Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := wrapWriter(w)
			next.ServeHTTP(rec, r)

			method, route := methodLabel(r.Method), RouteLabel(r)
			m.HTTPRequests.WithLabelValues(method, route, strconv.Itoa(rec.status)).Inc()
			m.HTTPDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
		})
	}
}
