package cmd

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by status code and HTTP method.",
		},
		[]string{"method", "code"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response latency (seconds) of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for metrics recording.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

// metricsHandler wraps an http.Handler to record Prometheus metrics.
func metricsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := wrapResponseWriter(w)

		h.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		// If status is 0, it means WriteHeader wasn't called, which defaults to 200 OK
		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}

		httpRequestsTotal.WithLabelValues(r.Method, strconv.Itoa(status)).Inc()
		httpRequestDuration.WithLabelValues(r.Method).Observe(duration)
	})
}

// metricsEndpoint returns the global prometheus HTTP handler
func metricsEndpoint() http.Handler {
	return promhttp.Handler()
}
