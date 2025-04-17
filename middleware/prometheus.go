package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"wellness-step-by-step/step-08/monitoring"
)

func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath() // или c.Request.URL.Path

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()

		// Записываем метрики
		monitoring.RequestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			http.StatusText(status),
		).Inc()

		monitoring.RequestDuration.WithLabelValues(
			c.Request.Method,
			path,
		).Observe(duration)
	}
}
