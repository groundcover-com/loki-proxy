package middlewares

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	URL_LABEL_NAME         = "url"
	METHOD_LABEL_NAME      = "method"
	STATUS_CODE_LABEL_NAME = "status_code"
	METRICS_ENDPOINT       = "/metrics"
)

var (
	requestDurationMetric = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_requests_duration_seconds",
			Help:    "Router request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{URL_LABEL_NAME, METHOD_LABEL_NAME, STATUS_CODE_LABEL_NAME},
	)

	requestTotalMetric = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of http requests",
		},
		[]string{URL_LABEL_NAME, METHOD_LABEL_NAME, STATUS_CODE_LABEL_NAME},
	)
)

func MetricsMiddleware(c *gin.Context) {
	path := c.FullPath()

	if path == METRICS_ENDPOINT {
		c.Next()
		return
	}

	start := time.Now()

	c.Next()

	status := strconv.Itoa(c.Writer.Status())
	elapsed := float64(time.Since(start)) / float64(time.Second)

	requestTotalMetric.WithLabelValues(path, c.Request.Method, status).Inc()
	requestDurationMetric.WithLabelValues(path, c.Request.Method, status).Observe(elapsed)
}
