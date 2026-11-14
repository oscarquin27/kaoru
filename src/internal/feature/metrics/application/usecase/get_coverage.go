package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/metrics/domain"
)

// GetCoverage devuelve la cobertura REQ por REQ, lo que hay que atender primero.
type GetCoverage struct {
	d Deps
}

// GetCoverageInput es la entrada de GetCoverage. Status vacío no filtra.
type GetCoverageInput struct {
	Filter domain.CoverageFilter
	Status string
	Page   int
	Size   int
}

// GetCoverageOutput es una página de la cobertura con el resumen de todo lo filtrado sin mirar Status.
type GetCoverageOutput struct {
	Summary domain.CoverageSummary
	Rows    []domain.CoverageRow
	Total   int
}

// Execute lee la cobertura. Leer no exige permiso.
func (uc *GetCoverage) Execute(ctx context.Context, in GetCoverageInput) (GetCoverageOutput, error) {
	if in.Status != "" && !domain.ValidCoverageStatus(in.Status) {
		return GetCoverageOutput{}, axis.Invalid(axis.InvalidField("status", "El estado debe ser below, no_test_cases, unassessed o meets."))
	}
	rows, err := uc.d.Repository.Coverage(ctx, in.Filter)
	if err != nil {
		return GetCoverageOutput{}, err
	}
	out := GetCoverageOutput{Summary: domain.SummarizeCoverage(rows)}
	if in.Status != "" {
		kept := rows[:0]
		for _, r := range rows {
			if r.Status() == domain.CoverageStatus(in.Status) {
				kept = append(kept, r)
			}
		}
		rows = kept
	}
	domain.SortCoverage(rows)
	out.Total, out.Rows = len(rows), page(rows, in.Page, in.Size)
	return out, nil
}
