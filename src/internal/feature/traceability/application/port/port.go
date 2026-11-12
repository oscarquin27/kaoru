// Package port declara lo que necesitan los casos de uso de traceability.
package port

import (
	"context"

	"kaoru/internal/feature/traceability/domain"
)

// Repository lee, sin escribir nunca, las tablas de todos los features.
type Repository interface {
	Systems(ctx context.Context) ([]domain.System, error)
	// RequirementHeader devuelve NOT_FOUND_RESOURCE si el REQ no existe.
	RequirementHeader(ctx context.Context, code string) (domain.RequirementHeader, error)
	// LifecycleStates devuelve las etapas con posición del flujo del REQ, en orden.
	LifecycleStates(ctx context.Context) ([]domain.StateRef, error)
	Transitions(ctx context.Context, requirementId string) ([]domain.Transition, error)
	FunctionalCases(ctx context.Context, requirementId string) ([]domain.CodeTitle, error)
	TestCases(ctx context.Context, requirementId string) ([]domain.TestCaseSummary, error)
	Defects(ctx context.Context, requirementId string) ([]domain.DefectSummary, error)

	Matrix(ctx context.Context, filter domain.MatrixFilter, page, size int) ([]domain.MatrixRow, error)
	// MatrixIndicators devuelve los indicadores y el total de filas de todo lo filtrado.
	MatrixIndicators(ctx context.Context, filter domain.MatrixFilter) (indicators domain.Indicators, totalRows int, err error)

	Search(ctx context.Context, query string, limit int) ([]domain.SearchHit, error)

	// PackageTrace devuelve la trazabilidad, los RUN y los DEF no cerrados de un PKG; vacíos si no existe.
	PackageTrace(ctx context.Context, packageCode string) (domain.PackageTrace, error)
}

// Readiness pregunta a requirement qué falta para que un REQ avance.
type Readiness interface {
	Of(ctx context.Context, requirementCode string) ([]domain.Readiness, error)
}

// Coverage pregunta a execution la cobertura de un REQ.
type Coverage interface {
	Of(ctx context.Context, requirementCode string) (domain.Coverage, error)
}
