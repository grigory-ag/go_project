package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ServerMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestsFailed  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func NewServerMetrics(registerer prometheus.Registerer) *ServerMetrics {
	metrics := &ServerMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		requestsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_failed_total",
				Help: "Total number of HTTP requests completed with an error status.",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}

	registerer.MustRegister(metrics.requestsTotal, metrics.requestsFailed, metrics.requestDuration)
	return metrics
}

func (m *ServerMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		writer := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(writer, r)

		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}

		statusLabel := strconv.Itoa(status)
		path := r.URL.Path

		m.requestsTotal.WithLabelValues(r.Method, path, statusLabel).Inc()
		m.requestDuration.WithLabelValues(r.Method, path).Observe(time.Since(startedAt).Seconds())
		if status >= http.StatusBadRequest {
			m.requestsFailed.WithLabelValues(r.Method, path, statusLabel).Inc()
		}
	})
}
