package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests."},
		[]string{"method", "path"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "Request duration in seconds."},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(RequestCounter, RequestDuration)
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		RequestCounter.WithLabelValues(c.Request.Method, c.FullPath()).Inc()
		RequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}
