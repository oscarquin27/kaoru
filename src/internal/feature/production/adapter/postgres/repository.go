// Package postgres implementa los puertos de production con sqlc.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/production/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee plantillas, CHK, DIAG y SMOKE.
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

// FindSystem devuelve NOT_FOUND_RESOURCE si el sistema no existe.
func (r *Repository) FindSystem(ctx context.Context, key string) (domain.System, error) {
	row, err := r.queries(ctx).GetSystem(ctx, key)
	if err != nil {
		return domain.System{}, corepg.TranslateError(err)
	}
	return domain.System{Key: row.Key, Name: row.Name, DefaultRoute: row.DefaultRoute}, nil
}

// LockPackage bloquea el PKG hasta el fin de la transacción en curso.
func (r *Repository) LockPackage(ctx context.Context, packageId string) error {
	id, err := parseID(packageId)
	if err != nil {
		return err
	}
	_, err = r.queries(ctx).LockPackage(ctx, id)
	return corepg.TranslateError(err)
}

// NextTemplateVersion devuelve la versión que sigue a la última publicada del sistema.
func (r *Repository) NextTemplateVersion(ctx context.Context, systemKey string) (int, error) {
	version, err := r.queries(ctx).NextTemplateVersion(ctx, systemKey)
	return int(version), corepg.TranslateError(err)
}

// InsertTemplate guarda una versión con sus pasos.
func (r *Repository) InsertTemplate(ctx context.Context, template domain.Template) error {
	s := template.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	q := r.queries(ctx)
	if err := q.InsertTemplate(ctx, sqlcgen.InsertTemplateParams{
		ID: id, SystemKey: s.System.Key, Version: int32(s.Version), //nolint:gosec // una versión cabe en int32
		CreatedBy: string(s.CreatedBy.Id), CreatedAt: s.CreatedAt,
	}); err != nil {
		return corepg.TranslateError(err)
	}
	for _, step := range s.Steps {
		if err := q.InsertTemplateStep(ctx, sqlcgen.InsertTemplateStepParams{
			TemplateID: id, Position: int32(step.Position), Title: step.Title, Description: step.Description, //nolint:gosec // acotado por el cuerpo
		}); err != nil {
			return corepg.TranslateError(err)
		}
	}
	return nil
}

// LatestTemplate devuelve NOT_FOUND_RESOURCE si el sistema no tiene plantilla.
func (r *Repository) LatestTemplate(ctx context.Context, systemKey string) (domain.Template, error) {
	row, err := r.queries(ctx).GetLatestTemplate(ctx, systemKey)
	if err != nil {
		return domain.Template{}, corepg.TranslateError(err)
	}
	templates, err := r.withSteps(ctx, []sqlcgen.ListTemplatesRow{sqlcgen.ListTemplatesRow(row)})
	if err != nil {
		return domain.Template{}, err
	}
	return templates[0], nil
}

// ListTemplates devuelve todas las versiones, la más reciente de cada sistema primero.
func (r *Repository) ListTemplates(ctx context.Context, systemKey string) ([]domain.Template, error) {
	rows, err := r.queries(ctx).ListTemplates(ctx, nullable(systemKey))
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	return r.withSteps(ctx, rows)
}

func (r *Repository) withSteps(ctx context.Context, rows []sqlcgen.ListTemplatesRow) ([]domain.Template, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	steps, err := r.templateSteps(ctx, ids)
	if err != nil {
		return nil, err
	}
	templates := make([]domain.Template, len(rows))
	for i, row := range rows {
		templates[i] = domain.RehydrateTemplate(domain.TemplateSnapshot{
			Id:      row.ID.String(),
			System:  domain.System{Key: row.SystemKey, Name: row.SystemName, DefaultRoute: row.SystemDefaultRoute},
			Version: int(row.Version), Steps: steps[row.ID],
			CreatedBy: domain.UserRef{Id: shared.UserId(row.CreatedBy), Name: row.CreatedByName}, CreatedAt: row.CreatedAt.UTC(),
		})
	}
	return templates, nil
}

// templateSteps devuelve los pasos de esas versiones, por plantilla y en orden.
func (r *Repository) templateSteps(ctx context.Context, templateIds []uuid.UUID) (map[uuid.UUID][]domain.TemplateStep, error) {
	rows, err := r.queries(ctx).ListTemplateSteps(ctx, templateIds)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	steps := make(map[uuid.UUID][]domain.TemplateStep)
	for _, row := range rows {
		steps[row.TemplateID] = append(steps[row.TemplateID], domain.TemplateStep{
			Position: int(row.Position), Title: row.Title, Description: row.Description,
		})
	}
	return steps, nil
}

// FindPackage devuelve NOT_FOUND_RESOURCE si el PKG no existe.
func (r *Repository) FindPackage(ctx context.Context, code string) (domain.PackageRef, error) {
	row, err := r.queries(ctx).GetPackageRef(ctx, code)
	if err != nil {
		return domain.PackageRef{}, corepg.TranslateError(err)
	}
	return domain.PackageRef{Id: row.ID.String(), Code: row.Code, Title: row.Title, SystemKey: row.SystemKey}, nil
}

// FindEnvironment devuelve NOT_FOUND_RESOURCE si el ENV no existe.
func (r *Repository) FindEnvironment(ctx context.Context, code string) (domain.EnvironmentRef, error) {
	row, err := r.queries(ctx).GetEnvironmentRef(ctx, code)
	if err != nil {
		return domain.EnvironmentRef{}, corepg.TranslateError(err)
	}
	return domain.EnvironmentRef{
		Id: row.ID.String(), Code: row.Code, Name: row.Name, Kind: row.Kind, SystemKey: row.SystemKey, Retired: row.Retired,
	}, nil
}

// InsertChecklist guarda un CHK. Sus pasos son los de la versión de la plantilla que referencia.
func (r *Repository) InsertChecklist(ctx context.Context, checklist domain.Checklist) error {
	s := checklist.Snapshot()
	ids, err := parseIDs(s.Id, s.Package.Id, s.Environment.Id, s.TemplateId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertChecklist(ctx, sqlcgen.InsertChecklistParams{
		ID: ids[0], Code: s.Code.String(), PackageID: ids[1], EnvironmentID: ids[2], TemplateID: ids[3],
		StartedBy: string(s.StartedBy.Id), CreatedAt: s.CreatedAt,
	}))
}

// InsertSignature guarda la firma de un paso. No existe una operación para cambiarla ni quitarla.
func (r *Repository) InsertSignature(ctx context.Context, checklistId string, signature domain.Signature) error {
	id, err := parseID(checklistId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertSignature(ctx, sqlcgen.InsertSignatureParams{
		ChecklistID: id, Position: int32(signature.Position), Comment: signature.Comment, //nolint:gosec // es un paso de la plantilla
		SignedBy: string(signature.SignedBy.Id), SignedAt: signature.SignedAt,
	}))
}

// FindChecklist devuelve NOT_FOUND_RESOURCE si el CHK no existe.
func (r *Repository) FindChecklist(ctx context.Context, code shared.Code) (domain.Checklist, error) {
	row, err := r.queries(ctx).GetChecklistByCode(ctx, code.String())
	if err != nil {
		return domain.Checklist{}, corepg.TranslateError(err)
	}
	checklists, err := r.toChecklists(ctx, []sqlcgen.ListChecklistsRow{sqlcgen.ListChecklistsRow(row)})
	if err != nil {
		return domain.Checklist{}, err
	}
	return checklists[0], nil
}

// ListChecklists devuelve una página de CHK, del más reciente al más antiguo.
func (r *Repository) ListChecklists(ctx context.Context, f port.Filter) ([]domain.Checklist, int, error) {
	q := r.queries(ctx)
	total, err := q.CountChecklists(ctx, nullable(f.PackageCode))
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListChecklists(ctx, sqlcgen.ListChecklistsParams{
		PackageCode: nullable(f.PackageCode),
		PageSize:    int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	checklists, err := r.toChecklists(ctx, rows)
	return checklists, int(total), err
}

// toChecklists arma los CHK con los pasos de su plantilla y las firmas que ya tienen.
func (r *Repository) toChecklists(ctx context.Context, rows []sqlcgen.ListChecklistsRow) ([]domain.Checklist, error) {
	ids, templateIds := make([]uuid.UUID, len(rows)), make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i], templateIds[i] = row.ID, row.TemplateID
	}
	steps, err := r.templateSteps(ctx, templateIds)
	if err != nil {
		return nil, err
	}
	signed, err := r.queries(ctx).ListSignatures(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	type key struct {
		checklist uuid.UUID
		position  int
	}
	signatures := make(map[key]*domain.Signature, len(signed))
	for _, g := range signed {
		signatures[key{g.ChecklistID, int(g.Position)}] = &domain.Signature{
			Position: int(g.Position), Comment: g.Comment,
			SignedBy: domain.UserRef{Id: shared.UserId(g.SignedBy), Name: g.SignedByName}, SignedAt: g.SignedAt.UTC(),
		}
	}
	checklists := make([]domain.Checklist, len(rows))
	for i, row := range rows {
		code, err := shared.ParseCode(row.Code)
		if err != nil {
			return nil, axis.Wrap(axis.InternalServerError, "código de CHK no válido en la base", err)
		}
		templateSteps := steps[row.TemplateID]
		checklistSteps := make([]domain.Step, len(templateSteps))
		for j, step := range templateSteps {
			checklistSteps[j] = domain.Step{TemplateStep: step, Signature: signatures[key{row.ID, step.Position}]}
		}
		checklists[i] = domain.RehydrateChecklist(domain.ChecklistSnapshot{
			Id: row.ID.String(), Code: code,
			Package: domain.PackageRef{Id: row.PackageID.String(), Code: row.PackageCode, Title: row.PackageTitle},
			Environment: domain.EnvironmentRef{
				Id: row.EnvironmentID.String(), Code: row.EnvironmentCode, Name: row.EnvironmentName, Kind: row.EnvironmentKind,
			},
			TemplateId: row.TemplateID.String(), TemplateVersion: int(row.TemplateVersion), Steps: checklistSteps,
			StartedBy: domain.UserRef{Id: shared.UserId(row.StartedBy), Name: row.StartedByName}, CreatedAt: row.CreatedAt.UTC(),
		})
	}
	return checklists, nil
}

// FindDeployment devuelve NOT_FOUND_RESOURCE si el DEP no existe.
func (r *Repository) FindDeployment(ctx context.Context, code string) (domain.DeploymentRef, error) {
	row, err := r.queries(ctx).GetDeploymentRef(ctx, code)
	if err != nil {
		return domain.DeploymentRef{}, corepg.TranslateError(err)
	}
	ref := deploymentRef(row.ID, row.Code, row.ExecutedAt, row.EnvironmentID, row.EnvironmentCode, row.EnvironmentName, row.EnvironmentKind)
	if row.PackageID.Valid && row.PackageCode != nil && row.PackageTitle != nil {
		ref.Package = &domain.PackageRef{Id: row.PackageID.UUID.String(), Code: *row.PackageCode, Title: *row.PackageTitle}
	}
	return ref, nil
}

// InsertDiagnostic guarda un DIAG.
func (r *Repository) InsertDiagnostic(ctx context.Context, diagnostic domain.Diagnostic) error {
	s := diagnostic.Snapshot()
	ids, err := parseIDs(s.Id, s.Deployment.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertDiagnostic(ctx, sqlcgen.InsertDiagnosticParams{
		ID: ids[0], Code: s.Code.String(), DeploymentID: ids[1], Notes: s.Notes,
		SubmittedBy: string(s.SubmittedBy.Id), CreatedAt: s.CreatedAt,
	}))
}

// InsertReview guarda la revisión de un DIAG. No existe una operación para cambiarla.
func (r *Repository) InsertReview(ctx context.Context, diagnosticId string, review domain.Review) error {
	id, err := parseID(diagnosticId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertDiagnosticReview(ctx, sqlcgen.InsertDiagnosticReviewParams{
		DiagnosticID: id, Result: string(review.Result), Notes: review.Notes,
		ReviewedBy: string(review.ReviewedBy.Id), ReviewedAt: review.ReviewedAt,
	}))
}

// FindDiagnostic devuelve NOT_FOUND_RESOURCE si el DIAG no existe.
func (r *Repository) FindDiagnostic(ctx context.Context, code shared.Code) (domain.Diagnostic, error) {
	row, err := r.queries(ctx).GetDiagnosticByCode(ctx, code.String())
	if err != nil {
		return domain.Diagnostic{}, corepg.TranslateError(err)
	}
	return toDiagnostic(sqlcgen.ListDiagnosticsRow(row))
}

// ListDiagnostics devuelve una página de DIAG, del más reciente al más antiguo.
func (r *Repository) ListDiagnostics(ctx context.Context, f port.Filter) ([]domain.Diagnostic, int, error) {
	q := r.queries(ctx)
	total, err := q.CountDiagnostics(ctx, nullable(f.PackageCode))
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListDiagnostics(ctx, sqlcgen.ListDiagnosticsParams{
		PackageCode: nullable(f.PackageCode),
		PageSize:    int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	diagnostics := make([]domain.Diagnostic, len(rows))
	for i, row := range rows {
		if diagnostics[i], err = toDiagnostic(row); err != nil {
			return nil, 0, err
		}
	}
	return diagnostics, int(total), nil
}

// InsertSmokeTest guarda un SMOKE. No existe una operación para editarlo ni borrarlo.
func (r *Repository) InsertSmokeTest(ctx context.Context, smokeTest domain.SmokeTest) error {
	s := smokeTest.Snapshot()
	ids, err := parseIDs(s.Id, s.Deployment.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertSmokeTest(ctx, sqlcgen.InsertSmokeTestParams{
		ID: ids[0], Code: s.Code.String(), DeploymentID: ids[1], Result: string(s.Result), Notes: s.Notes,
		ExecutedBy: string(s.ExecutedBy.Id), CreatedAt: s.CreatedAt,
	}))
}

// FindSmokeTest devuelve NOT_FOUND_RESOURCE si el SMOKE no existe.
func (r *Repository) FindSmokeTest(ctx context.Context, code shared.Code) (domain.SmokeTest, error) {
	row, err := r.queries(ctx).GetSmokeTestByCode(ctx, code.String())
	if err != nil {
		return domain.SmokeTest{}, corepg.TranslateError(err)
	}
	return toSmokeTest(sqlcgen.ListSmokeTestsRow(row))
}

// ListSmokeTests devuelve una página de SMOKE, del más reciente al más antiguo.
func (r *Repository) ListSmokeTests(ctx context.Context, f port.Filter) ([]domain.SmokeTest, int, error) {
	q := r.queries(ctx)
	total, err := q.CountSmokeTests(ctx, nullable(f.PackageCode))
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListSmokeTests(ctx, sqlcgen.ListSmokeTestsParams{
		PackageCode: nullable(f.PackageCode),
		PageSize:    int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	smokeTests := make([]domain.SmokeTest, len(rows))
	for i, row := range rows {
		if smokeTests[i], err = toSmokeTest(row); err != nil {
			return nil, 0, err
		}
	}
	return smokeTests, int(total), nil
}

// PackageFacts devuelve los hechos vacíos si el PKG no existe.
func (r *Repository) PackageFacts(ctx context.Context, packageCode string) (domain.PackageFacts, error) {
	row, err := r.queries(ctx).GetPackageFacts(ctx, packageCode)
	if err != nil {
		if err = corepg.TranslateError(err); axis.IsCode(err, axis.NotFoundResource) {
			return domain.PackageFacts{}, nil
		}
		return domain.PackageFacts{}, err
	}
	return domain.PackageFacts{
		ChecklistExists: row.ChecklistExists, ChecklistPending: int(row.ChecklistPending),
		Diagnostic: row.Diagnostic, Smoke: row.Smoke,
	}, nil
}

func deploymentRef(id uuid.UUID, code string, executedAt time.Time, envID uuid.UUID, envCode, envName, envKind string) domain.DeploymentRef {
	return domain.DeploymentRef{
		Id: id.String(), Code: code, ExecutedAt: executedAt.UTC(),
		Environment: domain.EnvironmentRef{Id: envID.String(), Code: envCode, Name: envName, Kind: envKind},
	}
}

func toDiagnostic(row sqlcgen.ListDiagnosticsRow) (domain.Diagnostic, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.Diagnostic{}, axis.Wrap(axis.InternalServerError, "código de DIAG no válido en la base", err)
	}
	s := domain.DiagnosticSnapshot{
		Id: row.ID.String(), Code: code,
		Package: domain.PackageRef{Id: row.PackageID.String(), Code: row.PackageCode, Title: row.PackageTitle},
		Deployment: deploymentRef(row.DeploymentID, row.DeploymentCode, row.DeploymentExecutedAt,
			row.EnvironmentID, row.EnvironmentCode, row.EnvironmentName, row.EnvironmentKind),
		Notes:       row.Notes,
		SubmittedBy: domain.UserRef{Id: shared.UserId(row.SubmittedBy), Name: row.SubmittedByName}, CreatedAt: row.CreatedAt.UTC(),
	}
	if row.ReviewResult != nil && row.ReviewNotes != nil && row.ReviewedBy != nil && row.ReviewedByName != nil && row.ReviewedAt != nil {
		s.Review = &domain.Review{
			Result: domain.ReviewResult(*row.ReviewResult), Notes: *row.ReviewNotes,
			ReviewedBy: domain.UserRef{Id: shared.UserId(*row.ReviewedBy), Name: *row.ReviewedByName}, ReviewedAt: row.ReviewedAt.UTC(),
		}
	}
	return domain.RehydrateDiagnostic(s), nil
}

func toSmokeTest(row sqlcgen.ListSmokeTestsRow) (domain.SmokeTest, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.SmokeTest{}, axis.Wrap(axis.InternalServerError, "código de SMOKE no válido en la base", err)
	}
	return domain.RehydrateSmokeTest(domain.SmokeTestSnapshot{
		Id: row.ID.String(), Code: code,
		Package: domain.PackageRef{Id: row.PackageID.String(), Code: row.PackageCode, Title: row.PackageTitle},
		Deployment: deploymentRef(row.DeploymentID, row.DeploymentCode, row.DeploymentExecutedAt,
			row.EnvironmentID, row.EnvironmentCode, row.EnvironmentName, row.EnvironmentKind),
		Result: domain.SmokeResult(row.Result), Notes: row.Notes,
		ExecutedBy: domain.UserRef{Id: shared.UserId(row.ExecutedBy), Name: row.ExecutedByName}, CreatedAt: row.CreatedAt.UTC(),
	}), nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}

func parseIDs(raw ...string) ([]uuid.UUID, error) {
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

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
