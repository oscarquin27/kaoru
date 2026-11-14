// Package postgres implementa las lecturas de metrics con sqlc. Ninguna consulta escribe.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/metrics/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/metrics/application/port"
	"kaoru/internal/feature/metrics/domain"
)

// Repository lee las tablas de los demás features.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// Coverage devuelve la cobertura de todos los REQ del filtro.
func (r *Repository) Coverage(ctx context.Context, f domain.CoverageFilter) ([]domain.CoverageRow, error) {
	rows, err := r.queries(ctx).ListCoverage(ctx, sqlcgen.ListCoverageParams{
		SystemKey: nullable(f.SystemKey), ChangeCode: nullable(f.ChangeCode), StateKey: nullable(f.StateKey),
		PackageCode: nullable(f.PackageCode),
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.CoverageRow, len(rows))
	for i, row := range rows {
		out[i] = domain.CoverageRow{
			Requirement: domain.CodeTitle{Code: row.Code, Title: row.Title}, SystemKey: row.SystemKey,
			State: domain.StateRef{Key: row.StateKey, Name: row.StateName}, RiskLevel: row.RiskLevel,
			Required: int(row.Required), Passed: int(row.Passed),
		}
		if row.MinimumPercent >= 0 {
			minimum := row.MinimumPercent
			out[i].MinimumPercent = &minimum
		}
	}
	return out, nil
}

// DefectStates devuelve los estados del flujo del DEF.
func (r *Repository) DefectStates(ctx context.Context) ([]domain.WorkflowState, error) {
	rows, err := r.queries(ctx).ListDefectStates(ctx)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.WorkflowState, len(rows))
	for i, row := range rows {
		out[i] = domain.WorkflowState{State: domain.StateRef{Key: row.Key, Name: row.Name}, IsFinal: row.IsFinal}
	}
	return out, nil
}

// DefectHistories devuelve los DEF del filtro con su historia, en dos consultas.
func (r *Repository) DefectHistories(ctx context.Context, f domain.DefectFilter) ([]domain.DefectHistory, error) {
	q := r.queries(ctx)
	defects, err := q.ListDefectsForStates(ctx, sqlcgen.ListDefectsForStatesParams{
		SystemKey: nullable(f.SystemKey), Severity: nullable(f.Severity), CreatedFrom: f.Period.From, CreatedTo: f.Period.To,
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	if len(defects) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(defects))
	index := make(map[uuid.UUID]int, len(defects))
	out := make([]domain.DefectHistory, len(defects))
	for i, d := range defects {
		ids[i], index[d.ID] = d.ID, i
		out[i] = domain.DefectHistory{Id: d.ID.String(), CreatedAt: d.CreatedAt.UTC(), Current: d.StateKey}
	}
	transitions, err := q.ListDefectTransitions(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	for _, t := range transitions {
		i := index[t.EntityID]
		out[i].Transitions = append(out[i].Transitions, domain.Transition{At: t.OccurredAt.UTC(), From: t.FromKey, To: t.ToKey})
	}
	return out, nil
}

// RegressionSizes devuelve los PKG con la regresión confirmada en el período.
func (r *Repository) RegressionSizes(ctx context.Context, f domain.RegressionFilter) ([]domain.RegressionRow, error) {
	from := f.Period.From
	rows, err := r.queries(ctx).ListRegressionSizes(ctx, sqlcgen.ListRegressionSizesParams{
		SystemKey: nullable(f.SystemKey), ConfirmedFrom: &from, ConfirmedTo: f.Period.To,
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.RegressionRow, len(rows))
	for i, row := range rows {
		out[i] = domain.RegressionRow{
			Package: domain.CodeTitle{Code: row.Code, Title: row.Title}, SystemKey: row.SystemKey,
			State: domain.StateRef{Key: row.StateKey, Name: row.StateName}, ConfirmedAt: row.ConfirmedAt.UTC(),
			Proposed: int(row.Proposed), Final: int(row.Final), Added: int(row.Added), Removed: int(row.Removed),
			Impact: int(row.Impact), Critical: int(row.Critical), Manual: int(row.Manual), Passed: int(row.Passed),
			SystemTestCases: int(row.SystemTestCases),
		}
	}
	return out, nil
}

// DefectReleases devuelve los DEF del filtro con la llegada a producción de su REQ.
func (r *Repository) DefectReleases(ctx context.Context, f domain.DefectFilter) ([]domain.DefectRelease, error) {
	rows, err := r.queries(ctx).ListDefectReleases(ctx, sqlcgen.ListDefectReleasesParams{
		SystemKey: nullable(f.SystemKey), Severity: nullable(f.Severity), CreatedFrom: f.Period.From, CreatedTo: f.Period.To,
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.DefectRelease, len(rows))
	for i, row := range rows {
		out[i] = domain.DefectRelease{
			Defect: domain.CodeTitle{Code: row.Code, Title: row.Title}, Severity: row.Severity,
			State: domain.StateRef{Key: row.StateKey, Name: row.StateName}, RequirementCode: row.RequirementCode,
			CreatedAt: row.CreatedAt.UTC(), PackageCode: row.PackageCode,
		}
		if row.ReleasedAt != "" {
			at, err := time.Parse(time.RFC3339Nano, row.ReleasedAt)
			if err != nil {
				return nil, fmt.Errorf("leer la llegada a producción de %s: %w", row.Code, err)
			}
			out[i].ReleasedAt = &at
		}
	}
	return out, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
