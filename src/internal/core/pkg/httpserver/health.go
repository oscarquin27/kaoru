package httpserver

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

// Health sirve /livez y /readyz en el puerto interno, que Nginx no publica.
type Health struct {
	ping     func(ctx context.Context) error
	draining atomic.Bool
}

// NewHealth recibe la comprobación de la base.
func NewHealth(ping func(ctx context.Context) error) *Health {
	return &Health{ping: ping}
}

// Drain hace que /readyz responda 503: el proceso se está apagando.
func (h *Health) Drain() { h.draining.Store(true) }

// Handler devuelve las rutas de salud.
func (h *Health) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if h.draining.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}
