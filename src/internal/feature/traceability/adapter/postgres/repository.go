// Package postgres implementa las lecturas de traceability con sqlc. Ninguna consulta escribe.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/traceability/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/traceability/application/port"
	"kaoru/internal/feature/traceability/domain"
)

const requirementEntity = "requirement"

// Repository lee las tablas de todos los features.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// Systems devuelve los sistemas bajo prueba.
func (r *Repository) Systems(ctx context.Context) ([]domain.System, error) {
	rows, err := r.queries(ctx).ListSystems(ctx)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.System, len(rows))
	for i, row := range rows {
		out[i] = domain.System{Key: row.Key, Name: row.Name, DefaultRoute: row.DefaultRoute}
	}
	return out, nil
}

// RequirementHeader devuelve NOT_FOUND_RESOURCE si el REQ no existe.
func (r *Repository) RequirementHeader(ctx context.Context, code string) (domain.RequirementHeader, error) {
	row, err := r.queries(ctx).GetRequirementHeader(ctx, code)
	if err != nil {
		return domain.RequirementHeader{}, corepg.TranslateError(err)
	}
	return domain.RequirementHeader{
		Id: row.ID.String(), Code: row.Code, Title: row.Title, Description: row.Description,
		AcceptanceCriteria: row.AcceptanceCriteria,
		System:             domain.CodeTitle{Code: row.SystemKey, Title: row.SystemName},
		Change:             domain.CodeTitle{Code: row.ChangeCode, Title: row.ChangeTitle},
		Route:              row.Route,
		State:              domain.StateRef{Key: row.StateKey, Name: row.StateName},
		Owner:              domain.UserRef{Id: row.OwnerID, Name: row.OwnerName},
		Risk:               domain.RiskSummary{Code: row.RiskCode, Level: row.RiskLevel, Justification: row.RiskJustification},
		CreatedAt:          row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}, nil
}

// LifecycleStates devuelve las etapas con posición del flujo del REQ, en orden.
func (r *Repository) LifecycleStates(ctx context.Context) ([]domain.StateRef, error) {
	rows, err := r.queries(ctx).ListLifecycleStates(ctx, requirementEntity)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.StateRef, len(rows))
	for i, row := range rows {
		out[i] = domain.StateRef{Key: row.Key, Name: row.Name}
	}
	return out, nil
}

// Transitions devuelve los cambios de estado de un REQ, en orden.
func (r *Repository) Transitions(ctx context.Context, requirementId string) ([]domain.Transition, error) {
	id, err := parseID(requirementId)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListTransitions(ctx, id)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.Transition, len(rows))
	for i, row := range rows {
		out[i] = domain.Transition{At: row.OccurredAt.UTC(), From: row.FromKey, To: row.ToKey}
	}
	return out, nil
}

// FunctionalCases devuelve los UC de un REQ.
func (r *Repository) FunctionalCases(ctx context.Context, requirementId string) ([]domain.CodeTitle, error) {
	id, err := parseID(requirementId)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListRequirementFunctionalCases(ctx, id)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.CodeTitle, len(rows))
	for i, row := range rows {
		out[i] = domain.CodeTitle{Code: row.Code, Title: row.Title}
	}
	return out, nil
}

// TestCases devuelve los TC de un REQ con su último RUN.
func (r *Repository) TestCases(ctx context.Context, requirementId string) ([]domain.TestCaseSummary, error) {
	id, err := parseID(requirementId)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListRequirementTestCases(ctx, id)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.TestCaseSummary, len(rows))
	for i, row := range rows {
		run, err := toRun(row.RunCode, row.RunResult, row.RunBuildVersion, row.RunEnvironmentName, row.RunExecutedAt)
		if err != nil {
			return nil, err
		}
		out[i] = domain.TestCaseSummary{Code: row.Code, Title: row.Title, IsCritical: row.IsCritical, Retired: row.Retired, LastRun: run}
	}
	return out, nil
}

// Defects devuelve los DEF de un REQ.
func (r *Repository) Defects(ctx context.Context, requirementId string) ([]domain.DefectSummary, error) {
	id, err := parseID(requirementId)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListRequirementDefects(ctx, id)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.DefectSummary, len(rows))
	for i, row := range rows {
		out[i] = domain.DefectSummary{
			Code: row.Code, Title: row.Title, Severity: row.Severity,
			State: domain.StateRef{Key: row.StateKey, Name: row.StateName}, IsFinal: row.IsFinal,
		}
	}
	return out, nil
}

// Matrix devuelve una página de filas de la matriz.
func (r *Repository) Matrix(ctx context.Context, f domain.MatrixFilter, page, size int) ([]domain.MatrixRow, error) {
	rows, err := r.queries(ctx).ListMatrix(ctx, sqlcgen.ListMatrixParams{
		ChangeCode: nullable(f.ChangeCode), SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey),
		RiskLevel: nullable(f.RiskLevel), OwnerID: nullable(f.OwnerId), CreatedFrom: f.CreatedFrom, CreatedTo: f.CreatedTo,
		PageSize: int32(size), PageOffset: int32((page - 1) * size), //nolint:gosec // acotados por el caso de uso
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.MatrixRow, len(rows))
	for i, row := range rows {
		run, err := toRun(row.RunCode, row.RunResult, row.RunBuildVersion, row.RunEnvironmentName, row.RunExecutedAt)
		if err != nil {
			return nil, err
		}
		var defects []struct {
			Code      string `json:"code"`
			Severity  string `json:"severity"`
			StateKey  string `json:"stateKey"`
			StateName string `json:"stateName"`
		}
		if err := json.Unmarshal(row.Defects, &defects); err != nil {
			return nil, fmt.Errorf("leer defectos de la matriz: %w", err)
		}
		out[i] = domain.MatrixRow{
			RequirementCode: row.RequirementCode, RequirementTitle: row.RequirementTitle, SystemKey: row.SystemKey,
			State:      domain.StateRef{Key: row.StateKey, Name: row.StateName},
			ChangeCode: row.ChangeCode, RiskCode: row.RiskCode, RiskLevel: row.RiskLevel, OwnerName: row.OwnerName,
			FunctionalCases: row.FunctionalCases,
			TestCase: domain.TestCaseSummary{
				Code: row.TestCaseCode, Title: row.TestCaseTitle, IsCritical: row.TestCaseCritical,
				Retired: row.TestCaseRetired, LastRun: run,
			},
			Defects: make([]domain.DefectSummary, len(defects)),
		}
		for j, d := range defects {
			out[i].Defects[j] = domain.DefectSummary{Code: d.Code, Severity: d.Severity, State: domain.StateRef{Key: d.StateKey, Name: d.StateName}}
		}
	}
	return out, nil
}

// MatrixIndicators devuelve los indicadores y el total de filas de todo lo filtrado.
func (r *Repository) MatrixIndicators(ctx context.Context, f domain.MatrixFilter) (domain.Indicators, int, error) {
	q := r.queries(ctx)
	row, err := q.GetMatrixIndicators(ctx, sqlcgen.GetMatrixIndicatorsParams{
		ChangeCode: nullable(f.ChangeCode), SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey),
		RiskLevel: nullable(f.RiskLevel), OwnerID: nullable(f.OwnerId), CreatedFrom: f.CreatedFrom, CreatedTo: f.CreatedTo,
	})
	if err != nil {
		return domain.Indicators{}, 0, corepg.TranslateError(err)
	}
	severities, err := q.CountOpenDefectsBySeverity(ctx, sqlcgen.CountOpenDefectsBySeverityParams{
		ChangeCode: nullable(f.ChangeCode), SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey),
		RiskLevel: nullable(f.RiskLevel), OwnerID: nullable(f.OwnerId), CreatedFrom: f.CreatedFrom, CreatedTo: f.CreatedTo,
	})
	if err != nil {
		return domain.Indicators{}, 0, corepg.TranslateError(err)
	}
	open := map[string]int{"high": 0, "medium": 0, "low": 0}
	for _, s := range severities {
		open[s.Severity] = int(s.Total)
	}
	return domain.NewIndicators(int(row.RequiredTestCases), int(row.PassedTestCases), int(row.BlockedRequirements), open),
		int(row.TotalRows), nil
}

// likeEscaper hace que %, _ y \ de lo que escribe el usuario se busquen como texto.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Search busca por ID o por texto del título en todas las entidades.
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]domain.SearchHit, error) {
	rows, err := r.queries(ctx).Search(ctx, sqlcgen.SearchParams{
		Query: query, Pattern: likeEscaper.Replace(query), MaxResults: int32(limit), //nolint:gosec // constante pequeña
	})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	out := make([]domain.SearchHit, len(rows))
	for i, row := range rows {
		out[i] = domain.SearchHit{Kind: row.Kind, Code: row.Code, Title: row.Title, State: domain.StateRef{Key: row.StateKey, Name: row.StateName}}
	}
	return out, nil
}

func toRun(code, result, build, environment, executedAt string) (domain.RunSummary, error) {
	if code == "" {
		return domain.RunSummary{}, nil
	}
	at, err := time.Parse(time.RFC3339Nano, executedAt)
	if err != nil {
		return domain.RunSummary{}, fmt.Errorf("leer fecha del RUN %s: %w", code, err)
	}
	return domain.RunSummary{Code: code, Result: result, BuildVersion: build, EnvironmentName: environment, ExecutedAt: at}, nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
