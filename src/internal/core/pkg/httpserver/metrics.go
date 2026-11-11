package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics son las métricas HTTP del servidor, siempre con las mismas etiquetas.
// `route` es la plantilla de la ruta (/api/requirements/:code), nunca la URL: así no crece sin límite.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewMetrics crea el registro con las métricas HTTP.
func NewMetrics() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_server_requests_total",
			Help: "Peticiones atendidas, por método, ruta y código de respuesta.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_server_request_duration_seconds",
			Help:    "Duración de las peticiones, por método y ruta.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
	}
	m.registry.MustRegister(m.requests, m.duration)
	return m
}

// Registry permite agregar otras métricas, como las del pool de Postgres.
func (m *Metrics) Registry() prometheus.Registerer { return m.registry }

// Middleware cuenta y mide cada petición.
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "sin_ruta"
		}
		m.requests.WithLabelValues(c.Request.Method, route, strconv.Itoa(c.Writer.Status())).Inc()
		m.duration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}

// Handler sirve las métricas en formato Prometheus. Va en el puerto interno.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
