package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: []float64{0.1, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path"},
	)
)

var (
	DatabaseQueries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total database queries",
		},
	)
)

func Init() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(RequestDuration)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
