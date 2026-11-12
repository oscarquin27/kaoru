// Package postgres implementa los puertos de execution con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/execution/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee RUN y EXP.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// NextCode reserva el siguiente ID legible del año para ese prefijo.
func (r *Repository) NextCode(ctx context.Context, prefix string, year int) (shared.Code, error) {
	seq, err := r.queries(ctx).NextCode(ctx, sqlcgen.NextCodeParams{Prefix: prefix, Year: int32(year)}) //nolint:gosec // un año cabe en int32
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(prefix, year, int(seq)), nil
}

// FindTestCase busca un TC por su código.
func (r *Repository) FindTestCase(ctx context.Context, code string) (domain.TestCaseRef, error) {
	row, err := r.queries(ctx).GetTestCaseRef(ctx, code)
	if err != nil {
		return domain.TestCaseRef{}, corepg.TranslateError(err)
	}
	return domain.TestCaseRef{
		Id: row.ID.String(), Code: row.Code, Title: row.Title, Retired: row.RetiredAt != nil, SystemKey: row.SystemKey,
	}, nil
}

// FindRequirement busca un REQ por su código.
func (r *Repository) FindRequirement(ctx context.Context, code string) (domain.RequirementRef, error) {
	row, err := r.queries(ctx).GetRequirementRef(ctx, code)
	if err != nil {
		return domain.RequirementRef{}, corepg.TranslateError(err)
	}
	return domain.RequirementRef{Id: row.ID.String(), Code: row.Code, SystemKey: row.SystemKey}, nil
}

// FindUser busca un usuario por su id.
func (r *Repository) FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error) {
	name, err := r.queries(ctx).GetUserName(ctx, string(id))
	if err != nil {
		return domain.UserRef{}, corepg.TranslateError(err)
	}
	return domain.UserRef{Id: id, Name: name}, nil
}

// InsertRun guarda un RUN. No existe una operación para editarlo ni borrarlo.
func (r *Repository) InsertRun(ctx context.Context, run domain.Run) error {
	s := run.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	testCaseID, err := parseID(s.TestCase.Id)
	if err != nil {
		return err
	}
	buildID, environmentID, err := parseCatalogIDs(s.BuildId, s.EnvironmentId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertRun(ctx, sqlcgen.InsertRunParams{
		ID: id, Code: s.Code.String(), TestCaseID: testCaseID, BuildVersion: s.BuildVersion,
		BuildChecksum: s.BuildChecksum, EnvironmentName: s.EnvironmentName, Result: string(s.Result),
		Notes: s.Notes, ExecutedBy: string(s.ExecutedBy.Id), ExecutedAt: s.ExecutedAt, CreatedAt: s.CreatedAt,
		BuildID: buildID, EnvironmentID: environmentID,
	}))
}

// FindRun busca un RUN por su ID legible.
func (r *Repository) FindRun(ctx context.Context, code shared.Code) (domain.Run, error) {
	row, err := r.queries(ctx).GetRunByCode(ctx, code.String())
	if err != nil {
		return domain.Run{}, corepg.TranslateError(err)
	}
	return toRun(sqlcgen.ListRunsRow(row))
}

// ListRuns devuelve una página de RUN, del más reciente al más antiguo.
func (r *Repository) ListRuns(ctx context.Context, f port.RunFilter) ([]domain.Run, int, error) {
	ids, err := parseIDs(f.TestCaseIds)
	if err != nil {
		return nil, 0, err
	}
	var build *string
	if f.BuildVersion != "" {
		build = &f.BuildVersion
	}
	q := r.queries(ctx)
	total, err := q.CountRuns(ctx, sqlcgen.CountRunsParams{FilterTestCases: f.FilterTestCases, TestCaseIds: ids, BuildVersion: build})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListRuns(ctx, sqlcgen.ListRunsParams{
		FilterTestCases: f.FilterTestCases, TestCaseIds: ids, BuildVersion: build,
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	runs := make([]domain.Run, len(rows))
	for i, row := range rows {
		if runs[i], err = toRun(row); err != nil {
			return nil, 0, err
		}
	}
	return runs, int(total), nil
}

// LatestResults devuelve el resultado del último RUN de cada TC que tenga alguno.
func (r *Repository) LatestResults(ctx context.Context, testCaseIds []string) (map[string]domain.Result, error) {
	ids, err := parseIDs(testCaseIds)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).LatestResults(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	latest := make(map[string]domain.Result, len(rows))
	for _, row := range rows {
		latest[row.TestCaseID.String()] = domain.Result(row.Result)
	}
	return latest, nil
}

// InsertExploration guarda un EXP nuevo.
func (r *Repository) InsertExploration(ctx context.Context, exploration domain.Exploration) error {
	s := exploration.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	requirementID, err := parseID(s.Requirement.Id)
	if err != nil {
		return err
	}
	buildID, environmentID, err := parseCatalogIDs(s.BuildId, s.EnvironmentId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertExploration(ctx, sqlcgen.InsertExplorationParams{
		ID: id, Code: s.Code.String(), RequirementID: requirementID, BuildVersion: s.BuildVersion,
		EnvironmentName: s.EnvironmentName, ManualChanges: s.ManualChanges, Findings: s.Findings,
		ExecutedBy: string(s.ExecutedBy.Id), ExecutedAt: s.ExecutedAt, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
		BuildID: buildID, EnvironmentID: environmentID,
	}))
}

// UpdateExploration guarda los cambios de un EXP.
func (r *Repository) UpdateExploration(ctx context.Context, exploration domain.Exploration) error {
	s := exploration.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateExploration(ctx, sqlcgen.UpdateExplorationParams{
		ID: id, ManualChanges: s.ManualChanges, Findings: s.Findings, UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "No existe.")
	}
	return nil
}

// FindExploration busca un EXP por su ID legible.
func (r *Repository) FindExploration(ctx context.Context, code shared.Code) (domain.Exploration, error) {
	row, err := r.queries(ctx).GetExplorationByCode(ctx, code.String())
	if err != nil {
		return domain.Exploration{}, corepg.TranslateError(err)
	}
	return toExploration(sqlcgen.ListExplorationsRow(row))
}

// ListExplorations devuelve una página de EXP; requirementCode vacío no filtra.
func (r *Repository) ListExplorations(ctx context.Context, requirementCode string, page, size int) ([]domain.Exploration, int, error) {
	var code *string
	if requirementCode != "" {
		code = &requirementCode
	}
	q := r.queries(ctx)
	total, err := q.CountExplorations(ctx, code)
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListExplorations(ctx, sqlcgen.ListExplorationsParams{
		RequirementCode: code, PageSize: int32(size), PageOffset: int32((page - 1) * size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	explorations := make([]domain.Exploration, len(rows))
	for i, row := range rows {
		if explorations[i], err = toExploration(row); err != nil {
			return nil, 0, err
		}
	}
	return explorations, int(total), nil
}

func toRun(row sqlcgen.ListRunsRow) (domain.Run, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.Run{}, axis.Wrap(axis.InternalServerError, "código de RUN no válido en la base", err)
	}
	return domain.RehydrateRun(domain.RunSnapshot{
		Id: row.ID.String(), Code: code,
		TestCase:     domain.TestCaseRef{Id: row.TestCaseID.String(), Code: row.TestCaseCode, Title: row.TestCaseTitle},
		BuildVersion: row.BuildVersion, BuildChecksum: row.BuildChecksum, EnvironmentName: row.EnvironmentName,
		BuildId: nullableID(row.BuildID), BuildCode: text(row.BuildCode),
		EnvironmentId: nullableID(row.EnvironmentID), EnvironmentCode: text(row.EnvironmentCode),
		Result: domain.Result(row.Result), Notes: row.Notes,
		ExecutedBy: domain.UserRef{Id: shared.UserId(row.ExecutedBy), Name: row.ExecutedByName},
		ExecutedAt: row.ExecutedAt.UTC(), CreatedAt: row.CreatedAt.UTC(),
	}), nil
}

func toExploration(row sqlcgen.ListExplorationsRow) (domain.Exploration, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.Exploration{}, axis.Wrap(axis.InternalServerError, "código de EXP no válido en la base", err)
	}
	return domain.RehydrateExploration(domain.ExplorationSnapshot{
		Id: row.ID.String(), Code: code,
		Requirement:  domain.RequirementRef{Id: row.RequirementID.String(), Code: row.RequirementCode},
		BuildVersion: row.BuildVersion, EnvironmentName: row.EnvironmentName,
		BuildId: nullableID(row.BuildID), BuildCode: text(row.BuildCode),
		EnvironmentId: nullableID(row.EnvironmentID), EnvironmentCode: text(row.EnvironmentCode),
		ManualChanges: row.ManualChanges, Findings: row.Findings,
		ExecutedBy: domain.UserRef{Id: shared.UserId(row.ExecutedBy), Name: row.ExecutedByName},
		ExecutedAt: row.ExecutedAt.UTC(), CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}), nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}

// parseCatalogIDs convierte los ids del BUILD y del ENV sobre los que se registra.
func parseCatalogIDs(build, environment string) (buildID, environmentID uuid.NullUUID, err error) {
	if buildID.UUID, err = parseID(build); err != nil {
		return buildID, environmentID, err
	}
	if environmentID.UUID, err = parseID(environment); err != nil {
		return buildID, environmentID, err
	}
	buildID.Valid, environmentID.Valid = true, true
	return buildID, environmentID, nil
}

// nullableID y text devuelven vacío en las filas anteriores al catálogo, que no tienen BUILD ni ENV.
func nullableID(id uuid.NullUUID) string {
	if !id.Valid {
		return ""
	}
	return id.UUID.String()
}

func text(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseIDs(raw []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(raw))
	for i, r := range raw {
		id, err := parseID(r)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}
