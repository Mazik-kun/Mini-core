package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Tracing создаёт серверный span на запрос (имя — "METHOD /route/template").
func Tracing() Middleware {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "gateway",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return r.Method + " " + RouteLabel(r)
			}))
	}
}
