// Package port declara lo que necesitan los casos de uso de metrics.
package port

import (
	"context"

	"kaoru/internal/feature/metrics/domain"
)

// Repository lee, sin escribir nunca, las tablas de los demás features.
type Repository interface {
	// Coverage devuelve la cobertura de todos los REQ del filtro, sin orden.
	Coverage(ctx context.Context, filter domain.CoverageFilter) ([]domain.CoverageRow, error)
	// DefectStates devuelve los estados del flujo del DEF en el orden de su línea de vida.
	DefectStates(ctx context.Context) ([]domain.WorkflowState, error)
	// DefectHistories devuelve los DEF del filtro con sus transiciones en orden.
	DefectHistories(ctx context.Context, filter domain.DefectFilter) ([]domain.DefectHistory, error)
	// RegressionSizes devuelve los PKG con la regresión confirmada en el período, la más reciente primero.
	RegressionSizes(ctx context.Context, filter domain.RegressionFilter) ([]domain.RegressionRow, error)
	// DefectReleases devuelve los DEF del filtro, el más reciente primero, con la llegada a producción de su REQ.
	DefectReleases(ctx context.Context, filter domain.DefectFilter) ([]domain.DefectRelease, error)
}
