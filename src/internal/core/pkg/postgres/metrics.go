package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// RegisterPoolMetrics publica las estadísticas del pool de conexiones.
func RegisterPoolMetrics(registry prometheus.Registerer, pool *pgxpool.Pool) error {
	gauges := map[string]struct {
		help  string
		value func(*pgxpool.Stat) float64
	}{
		"db_pool_acquired_connections": {"Conexiones en uso.", func(s *pgxpool.Stat) float64 { return float64(s.AcquiredConns()) }},
		"db_pool_idle_connections":     {"Conexiones libres.", func(s *pgxpool.Stat) float64 { return float64(s.IdleConns()) }},
		"db_pool_total_connections":    {"Conexiones abiertas.", func(s *pgxpool.Stat) float64 { return float64(s.TotalConns()) }},
		"db_pool_max_connections":      {"Tope de conexiones del pool.", func(s *pgxpool.Stat) float64 { return float64(s.MaxConns()) }},
	}
	for name, g := range gauges {
		collector := prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: name, Help: g.help}, func() float64 { return g.value(pool.Stat()) })
		if err := registry.Register(collector); err != nil {
			return err
		}
	}
	return nil
}
