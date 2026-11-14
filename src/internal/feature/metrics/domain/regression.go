package domain

import "time"

// RegressionFilter acota los PKG por la fecha en que se confirmó su regresión.
type RegressionFilter struct {
	SystemKey string
	Period    Period
}

// RegressionRow es el tamaño de la regresión confirmada de un PKG.
type RegressionRow struct {
	Package     CodeTitle
	SystemKey   string
	State       StateRef
	ConfirmedAt time.Time
	// Proposed son los TC que propuso el sistema; Final, los que quedaron.
	Proposed int
	Final    int
	// Added son los que QA agregó a lo propuesto; Removed, los propuestos que quitó.
	Added   int
	Removed int
	// Los TC finales por origen.
	Impact   int
	Critical int
	Manual   int
	// Passed son los TC finales con su último RUN sobre un BUILD del PKG aprobado.
	Passed int
	// SystemTestCases son los TC activos del sistema hoy.
	SystemTestCases int
}

// RegressionSummary suma la regresión de varios PKG.
type RegressionSummary struct {
	Packages    int
	Proposed    int
	Final       int
	Added       int
	Removed     int
	MedianFinal *float64
}

// SummarizeRegression resume todas las filas.
func SummarizeRegression(rows []RegressionRow) RegressionSummary {
	s := RegressionSummary{Packages: len(rows)}
	finals := make([]float64, len(rows))
	for i, r := range rows {
		s.Proposed += r.Proposed
		s.Final += r.Final
		s.Added += r.Added
		s.Removed += r.Removed
		finals[i] = float64(r.Final)
	}
	s.MedianFinal = Median(finals)
	return s
}
