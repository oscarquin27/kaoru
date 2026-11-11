// Package postgres implementa los puertos de testdesign con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/testdesign/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los UC y los TC.
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

// FindSystem devuelve el nombre del sistema.
func (r *Repository) FindSystem(ctx context.Context, key string) (string, error) {
	row, err := r.queries(ctx).GetSystem(ctx, key)
	if err != nil {
		return "", corepg.TranslateError(err)
	}
	return row.Name, nil
}

// InsertFunctionalCase guarda un UC nuevo.
func (r *Repository) InsertFunctionalCase(ctx context.Context, functionalCase domain.FunctionalCase) error {
	s := functionalCase.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	requirementID, err := parseID(s.Requirement.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertFunctionalCase(ctx, sqlcgen.InsertFunctionalCaseParams{
		ID: id, Code: s.Code.String(), RequirementID: requirementID, Title: s.Title, Description: s.Description,
		CreatedBy: string(s.CreatedBy), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// UpdateFunctionalCase guarda los cambios de un UC.
func (r *Repository) UpdateFunctionalCase(ctx context.Context, functionalCase domain.FunctionalCase) error {
	s := functionalCase.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateFunctionalCase(ctx, sqlcgen.UpdateFunctionalCaseParams{
		ID: id, Title: s.Title, Description: s.Description, UpdatedAt: s.UpdatedAt,
	})
	return affected(rows, err)
}

// FindFunctionalCase busca un UC por su ID legible, con los TC que lo validan.
func (r *Repository) FindFunctionalCase(ctx context.Context, code shared.Code) (domain.FunctionalCase, error) {
	row, err := r.queries(ctx).GetFunctionalCaseByCode(ctx, code.String())
	if err != nil {
		return domain.FunctionalCase{}, corepg.TranslateError(err)
	}
	functionalCases, err := r.toFunctionalCases(ctx, []sqlcgen.ListFunctionalCasesByRequirementRow{sqlcgen.ListFunctionalCasesByRequirementRow(row)})
	if err != nil {
		return domain.FunctionalCase{}, err
	}
	return functionalCases[0], nil
}

// ListFunctionalCases devuelve los UC de un REQ en orden de código.
func (r *Repository) ListFunctionalCases(ctx context.Context, requirementId string) ([]domain.FunctionalCase, error) {
	requirementID, err := parseID(requirementId)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListFunctionalCasesByRequirement(ctx, requirementID)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	return r.toFunctionalCases(ctx, rows)
}

func (r *Repository) toFunctionalCases(ctx context.Context, rows []sqlcgen.ListFunctionalCasesByRequirementRow) ([]domain.FunctionalCase, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	links, err := r.queries(ctx).ListTestCaseCodesByFunctionalCases(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	testCases := map[uuid.UUID][]string{}
	for _, link := range links {
		testCases[link.FunctionalCaseID] = append(testCases[link.FunctionalCaseID], link.Code)
	}
	out := make([]domain.FunctionalCase, len(rows))
	for i, row := range rows {
		code, err := shared.ParseCode(row.Code)
		if err != nil {
			return nil, axis.Wrap(axis.InternalServerError, "código de UC no válido en la base", err)
		}
		codes := testCases[row.ID]
		if codes == nil {
			codes = []string{}
		}
		out[i] = domain.RehydrateFunctionalCase(domain.FunctionalCaseSnapshot{
			Id: row.ID.String(), Code: code,
			Requirement: domain.RequirementRef{Id: row.RequirementID.String(), Code: row.RequirementCode, SystemKey: row.SystemKey},
			Title:       row.Title, Description: row.Description, TestCases: codes,
			CreatedBy: shared.UserId(row.CreatedBy), CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
		})
	}
	return out, nil
}

// FunctionalCaseRefs devuelve los UC que existen entre los códigos dados.
func (r *Repository) FunctionalCaseRefs(ctx context.Context, codes []string) ([]domain.FunctionalCaseRef, error) {
	rows, err := r.queries(ctx).ListFunctionalCaseRefsByCodes(ctx, codes)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	refs := make([]domain.FunctionalCaseRef, len(rows))
	for i, row := range rows {
		refs[i] = domain.FunctionalCaseRef{Id: row.ID.String(), Code: row.Code, SystemKey: row.SystemKey}
	}
	return refs, nil
}

// InsertTestCase guarda un TC nuevo con los UC que valida.
func (r *Repository) InsertTestCase(ctx context.Context, testCase domain.TestCase) error {
	s := testCase.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	if err := r.queries(ctx).InsertTestCase(ctx, sqlcgen.InsertTestCaseParams{
		ID: id, Code: s.Code.String(), SystemKey: s.SystemKey, Title: s.Title, Preconditions: s.Preconditions,
		Steps: s.Steps, ExpectedResult: s.ExpectedResult, IsCritical: s.IsCritical, RetiredAt: s.RetiredAt,
		CreatedBy: string(s.CreatedBy), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}); err != nil {
		return corepg.TranslateError(err)
	}
	return r.insertLinks(ctx, id, s.FunctionalCases)
}

// UpdateTestCase guarda el TC y reemplaza los UC que valida.
func (r *Repository) UpdateTestCase(ctx context.Context, testCase domain.TestCase) error {
	s := testCase.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	q := r.queries(ctx)
	rows, err := q.UpdateTestCase(ctx, sqlcgen.UpdateTestCaseParams{
		ID: id, Title: s.Title, Preconditions: s.Preconditions, Steps: s.Steps, ExpectedResult: s.ExpectedResult,
		IsCritical: s.IsCritical, RetiredAt: s.RetiredAt, UpdatedAt: s.UpdatedAt,
	})
	if err := affected(rows, err); err != nil {
		return err
	}
	if err := q.DeleteTestCaseLinks(ctx, id); err != nil {
		return corepg.TranslateError(err)
	}
	return r.insertLinks(ctx, id, s.FunctionalCases)
}

func (r *Repository) insertLinks(ctx context.Context, testCaseID uuid.UUID, refs []domain.FunctionalCaseRef) error {
	q := r.queries(ctx)
	for _, ref := range refs {
		functionalCaseID, err := parseID(ref.Id)
		if err != nil {
			return err
		}
		if err := q.InsertTestCaseLink(ctx, sqlcgen.InsertTestCaseLinkParams{TestCaseID: testCaseID, FunctionalCaseID: functionalCaseID}); err != nil {
			return corepg.TranslateError(err)
		}
	}
	return nil
}

// FindTestCase busca un TC por su ID legible.
func (r *Repository) FindTestCase(ctx context.Context, code shared.Code) (domain.TestCase, error) {
	row, err := r.queries(ctx).GetTestCaseByCode(ctx, code.String())
	if err != nil {
		return domain.TestCase{}, corepg.TranslateError(err)
	}
	testCases, err := r.toTestCases(ctx, []sqlcgen.ListTestCasesRow{sqlcgen.ListTestCasesRow(row)})
	if err != nil {
		return domain.TestCase{}, err
	}
	return testCases[0], nil
}

// ListTestCases devuelve una página de TC en orden de código.
func (r *Repository) ListTestCases(ctx context.Context, f port.TestCaseFilter) ([]domain.TestCase, int, error) {
	var requirementID uuid.NullUUID
	if f.RequirementId != "" {
		id, err := parseID(f.RequirementId)
		if err != nil {
			return nil, 0, err
		}
		requirementID = uuid.NullUUID{UUID: id, Valid: true}
	}
	q := r.queries(ctx)
	total, err := q.CountTestCases(ctx, sqlcgen.CountTestCasesParams{
		SystemKey: nullable(f.SystemKey), Critical: f.Critical, Query: nullable(f.Query), RequirementID: requirementID,
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListTestCases(ctx, sqlcgen.ListTestCasesParams{
		SystemKey: nullable(f.SystemKey), Critical: f.Critical, Query: nullable(f.Query), RequirementID: requirementID,
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	testCases, err := r.toTestCases(ctx, rows)
	return testCases, int(total), err
}

func (r *Repository) toTestCases(ctx context.Context, rows []sqlcgen.ListTestCasesRow) ([]domain.TestCase, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	links, err := r.queries(ctx).ListFunctionalCaseRefsByTestCases(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	refs := map[uuid.UUID][]domain.FunctionalCaseRef{}
	for _, link := range links {
		refs[link.TestCaseID] = append(refs[link.TestCaseID], domain.FunctionalCaseRef{Id: link.ID.String(), Code: link.Code, SystemKey: link.SystemKey})
	}
	out := make([]domain.TestCase, len(rows))
	for i, row := range rows {
		code, err := shared.ParseCode(row.Code)
		if err != nil {
			return nil, axis.Wrap(axis.InternalServerError, "código de TC no válido en la base", err)
		}
		s := domain.TestCaseSnapshot{
			Id: row.ID.String(), Code: code, SystemKey: row.SystemKey, SystemName: row.SystemName, Title: row.Title,
			Preconditions: row.Preconditions, Steps: row.Steps, ExpectedResult: row.ExpectedResult,
			IsCritical: row.IsCritical, FunctionalCases: refs[row.ID],
			CreatedBy: shared.UserId(row.CreatedBy), CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
		}
		if row.RetiredAt != nil {
			retired := row.RetiredAt.UTC()
			s.RetiredAt = &retired
		}
		out[i] = domain.RehydrateTestCase(s)
	}
	return out, nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}

func affected(rows int64, err error) error {
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "No existe.")
	}
	return nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
