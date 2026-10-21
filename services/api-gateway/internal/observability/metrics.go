// Package observability — Prometheus-метрики и OpenTelemetry-трейсинг.
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	Registry *prometheus.Registry

	HTTPRequests       *prometheus.CounterVec
	HTTPDuration       *prometheus.HistogramVec
	RLRejected         *prometheus.CounterVec
	RLRedisErrors      *prometheus.CounterVec
	JWKSRefresh        *prometheus.CounterVec
	GRPCClientDuration *prometheus.HistogramVec
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	f := promauto.With(reg)
	return &Metrics{
		Registry: reg,
		HTTPRequests: f.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total", Help: "HTTP requests by method, route template and status.",
		}, []string{"method", "route", "status"}),
		HTTPDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds", Help: "HTTP request latency.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		RLRejected: f.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_ratelimit_rejected_total", Help: "Requests rejected by rate limiter.",
		}, []string{"scope"}),
		RLRedisErrors: f.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_ratelimit_redis_errors_total", Help: "Redis errors in rate limiter (fail-closed).",
		}, []string{"op"}),
		JWKSRefresh: f.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_jwks_refresh_total", Help: "JWKS refresh attempts.",
		}, []string{"result"}),
		GRPCClientDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name: "gateway_grpc_client_duration_seconds", Help: "Downstream gRPC call latency.",
			Buckets: prometheus.DefBuckets,
		}, []string{"target", "method", "code"}),
	}
}
