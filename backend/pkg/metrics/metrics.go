// Package metrics provides Prometheus metrics for MediLink.
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "medilink_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "medilink_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	FHIROperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "medilink_fhir_operations_total",
			Help: "Total FHIR operations by resource type and action",
		},
		[]string{"resource_type", "action"},
	)

	DrugCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "medilink_drug_checks_total",
			Help: "Total drug interaction checks",
		},
		[]string{"result"},
	)

	DocumentJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "medilink_document_jobs_total",
			Help: "Total document processing jobs",
		},
		[]string{"status"},
	)
)

func init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		FHIROperationsTotal,
		DrugCheckTotal,
		DocumentJobsTotal,
	)
}

// GinMiddleware returns a Gin middleware that records HTTP metrics.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// Handler returns the Prometheus metrics HTTP handler for Gin.
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
