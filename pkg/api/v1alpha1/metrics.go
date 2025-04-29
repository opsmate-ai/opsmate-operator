package v1alpha1

import (
	"context"
	"net/http"

	"github.com/opsmate-ai/opsmate-operator/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Registry          = prometheus.NewRegistry()
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"code", "method", "path"},
	)
	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "method", "path"},
	)
)

func InitMetricsServer(ctx context.Context) {
	Registry.MustRegister(HttpRequestsTotal)
	Registry.MustRegister(HttpRequestDuration)
	Registry.MustRegister(collectors.NewGoCollector())

	http.Handle("/metrics", promhttp.HandlerFor(
		Registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	logger.G(ctx).Info("Starting metrics server on :8081")
	go func() {
		server := &http.Server{Addr: "0.0.0.0:8081"}

		go func() {
			<-ctx.Done()
			if err := server.Shutdown(context.Background()); err != nil {
				logger.G(ctx).WithError(err).Error("failed to gracefully shutdown metrics server")
			}
		}()

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// non-critical, just log the error
			logger.G(ctx).WithError(err).Error("failed to start metrics server")
		}
	}()
}
