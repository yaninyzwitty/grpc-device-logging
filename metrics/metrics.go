package metrics

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yaninyzwitty/grpc-device-logging/config"
)

// Metrics holds Prometheus collectors for the service.
type Metrics struct {
	Stage    prometheus.Gauge
	Duration *prometheus.HistogramVec
	Errors   *prometheus.CounterVec
}

// NewMetrics registers and returns a Metrics instance.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Stage: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "myapp",
			Name:      "stage",
			Help:      "Current stage of the application/test run",
		}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "myapp",
			Name:      "request_duration_seconds",
			Help:      "Duration of gRPC or DB requests in seconds",
			Buckets:   prometheus.DefBuckets, // standard buckets: 0.005 → 10s
		}, []string{"op", "db"}),
		Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "myapp",
			Name:      "errors_total",
			Help:      "Count of errors by operation and backend",
		}, []string{"op", "db"}),
	}

	// Register metrics with Prometheus
	reg.MustRegister(m.Stage, m.Duration, m.Errors)

	return m
}

// StartPrometheusServer launches an HTTP server for Prometheus scraping.
func StartPrometheusServer(c *config.Config, reg *prometheus.Registry) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	addr := fmt.Sprintf(":%d", c.MetricsPort)
	go func() {
		log.Printf("[metrics] Starting Prometheus server on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("[metrics] Prometheus server failed: %v", err)
		}
	}()
}
