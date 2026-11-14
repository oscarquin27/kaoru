// Package domain reúne los cálculos de las métricas. Este feature no tiene entidades ni escribe
// nada: cruza lo que guardan los demás, como traceability, y lo resume. No hay SLA (regla 3):
// las métricas miden, no califican.
package domain

import (
	"math"
	"slices"
	"time"
)

// StateRef es un estado con el nombre que se muestra.
type StateRef struct {
	Key  string
	Name string
}

// CodeTitle es un par de ID legible y título.
type CodeTitle struct {
	Code  string
	Title string
}

// Period acota por fecha: desde From inclusive hasta To exclusivo; To nil no tiene fin.
type Period struct {
	From time.Time
	To   *time.Time
}

// DefaultFrom es el inicio del período cuando no se pide uno: el primer día del mes de hace 11 meses,
// en UTC. Con el mes en curso suman 12 meses.
func DefaultFrom(now time.Time) time.Time {
	now = now.UTC()
	return time.Date(now.Year(), now.Month()-11, 1, 0, 0, 0, 0, time.UTC)
}

// Months devuelve los meses del período, en UTC y del más antiguo al más reciente, como "2026-11".
// Un período sin fin llega hasta el mes de now.
func (p Period) Months(now time.Time) []string {
	last := now.UTC()
	if p.To != nil {
		last = p.To.UTC().Add(-time.Nanosecond)
	}
	var out []string
	month := time.Date(p.From.UTC().Year(), p.From.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	for !month.After(last) {
		out = append(out, month.Format("2006-01"))
		month = month.AddDate(0, 1, 0)
	}
	return out
}

// Median es la mediana; con una cantidad par, el promedio de los dos del medio. nil sin valores.
func Median(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	sorted := slices.Sorted(slices.Values(values))
	mid := len(sorted) / 2
	m := sorted[mid]
	if len(sorted)%2 == 0 {
		m = (sorted[mid-1] + sorted[mid]) / 2
	}
	return &m
}

// Percentile es el percentil q (entre 0 y 1) por rango más cercano: el menor valor que deja
// por debajo o igual al menos la fracción q de los valores. nil sin valores.
func Percentile(values []float64, q float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	sorted := slices.Sorted(slices.Values(values))
	rank := int(math.Ceil(q*float64(len(sorted)))) - 1
	rank = max(0, min(rank, len(sorted)-1))
	v := sorted[rank]
	return &v
}

// Days pasa una duración a días naturales.
func Days(d time.Duration) float64 { return d.Hours() / 24 }
