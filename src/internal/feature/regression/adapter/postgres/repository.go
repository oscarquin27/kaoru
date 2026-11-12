// Package postgres implementa los puertos de regression con sqlc.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/regression/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/regression/application/port"
	"kaoru/internal/feature/regression/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda la selección de regresión y lee lo que necesita del PKG, los TC y los RUN.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// FindPackage devuelve NOT_FOUND_RESOURCE si el PKG no existe.
func (r *Repository) FindPackage(ctx context.Context, code shared.Code) (port.PackageRef, error) {
	row, err := r.queries(ctx).GetPackage(ctx, code.String())
	if err != nil {
		return port.PackageRef{}, corepg.TranslateError(err)
	}
	return port.PackageRef{Id: row.ID.String(), Code: code, SystemKey: row.SystemKey}, nil
}

// LockPackage bloquea el PKG hasta el fin de la transacción en curso y lo devuelve.
func (r *Repository) LockPackage(ctx context.Context, code shared.Code) (port.PackageRef, error) {
	if _, err := r.queries(ctx).LockPackage(ctx, code.String()); err != nil {
		return port.PackageRef{}, corepg.TranslateError(err)
	}
	return r.FindPackage(ctx, code)
}

// FindTestCase devuelve NOT_FOUND_RESOURCE si el TC no existe.
func (r *Repository) FindTestCase(ctx context.Context, code shared.Code) (domain.TestCase, error) {
	row, err := r.queries(ctx).GetTestCase(ctx, code.String())
	if err != nil {
		return domain.TestCase{}, corepg.TranslateError(err)
	}
	return domain.TestCase{
		Ref: domain.TestCaseRef{Id: row.ID.String(), Code: row.Code}, SystemKey: row.SystemKey, Retired: row.Retired,
	}, nil
}

// ListCandidates devuelve lo que el sistema propone para el PKG, por código de TC.
func (r *Repository) ListCandidates(ctx context.Context, packageId string) ([]domain.Candidate, error) {
	id, err := parseID(packageId)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries(ctx).ListCandidates(ctx, id)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	candidates := make([]domain.Candidate, len(rows))
	for i, row := range rows {
		origin := domain.OriginCritical
		if row.ByImpact {
			origin = domain.OriginImpact
		}
		candidates[i] = domain.Candidate{TestCase: domain.TestCaseRef{Id: row.ID.String(), Code: row.Code}, Origin: origin}
	}
	return candidates, nil
}

// FindSelection devuelve NOT_FOUND_RESOURCE si el PKG no tiene selección.
func (r *Repository) FindSelection(ctx context.Context, packageId string) (port.SelectionView, error) {
	id, err := parseID(packageId)
	if err != nil {
		return port.SelectionView{}, err
	}
	q := r.queries(ctx)
	header, err := q.GetSelection(ctx, id)
	if err != nil {
		return port.SelectionView{}, corepg.TranslateError(err)
	}
	view := port.SelectionView{
		Selection: domain.Selection{
			PackageId: packageId, ProposedBy: shared.UserId(header.ProposedBy), ProposedAt: header.ProposedAt.UTC(),
			ConfirmedBy: shared.UserId(text(header.ConfirmedBy)), ConfirmedAt: utc(header.ConfirmedAt),
		},
		ProposedByName: header.ProposedByName, ConfirmedByName: text(header.ConfirmedByName),
	}
	rows, err := q.ListItems(ctx, id)
	if err != nil {
		return port.SelectionView{}, corepg.TranslateError(err)
	}
	componentRows, err := q.ListItemComponents(ctx, id)
	if err != nil {
		return port.SelectionView{}, corepg.TranslateError(err)
	}
	components := make(map[uuid.UUID][]port.ComponentRef)
	for _, c := range componentRows {
		components[c.TestCaseID] = append(components[c.TestCaseID], port.ComponentRef{Code: c.Code, Name: c.Name, Kind: c.Kind})
	}
	view.Items = make([]port.ItemView, len(rows))
	for i, row := range rows {
		view.Items[i] = port.ItemView{
			Item: domain.Item{
				TestCase: domain.TestCaseRef{Id: row.TestCaseID.String(), Code: row.TestCaseCode},
				Origin:   domain.Origin(row.Origin), Proposed: row.Proposed, Included: row.Included, Reason: row.Reason,
				DecidedBy: shared.UserId(text(row.DecidedBy)), DecidedAt: utc(row.DecidedAt),
			},
			Title: row.TestCaseTitle, IsCritical: row.IsCritical, DecidedByName: text(row.DecidedByName),
			Components: components[row.TestCaseID], LastRun: lastRun(row),
		}
	}
	return view, nil
}

// lastRun devuelve nil si el TC no tiene un RUN sobre un BUILD del PKG.
func lastRun(row sqlcgen.ListItemsRow) *port.RunRef {
	if row.RunCode == nil || row.RunResult == nil || row.RunExecutedAt == nil {
		return nil
	}
	return &port.RunRef{
		Code: *row.RunCode, Result: *row.RunResult, BuildVersion: text(row.RunBuildVersion),
		EnvironmentName: text(row.RunEnvironmentName), BuildCode: text(row.RunBuildCode),
		EnvironmentCode: text(row.RunEnvironmentCode), ExecutedAt: row.RunExecutedAt.UTC(),
	}
}

// SaveSelection crea la cabecera o guarda sus cambios.
func (r *Repository) SaveSelection(ctx context.Context, s domain.Selection) error {
	id, err := parseID(s.PackageId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).UpsertSelection(ctx, sqlcgen.UpsertSelectionParams{
		PackageID: id, ProposedBy: string(s.ProposedBy), ProposedAt: s.ProposedAt,
		ConfirmedBy: nullable(string(s.ConfirmedBy)), ConfirmedAt: s.ConfirmedAt,
	}))
}

// ReplaceItems deja como TC de la selección exactamente los dados.
func (r *Repository) ReplaceItems(ctx context.Context, packageId string, items []domain.Item) error {
	id, err := parseID(packageId)
	if err != nil {
		return err
	}
	if err := r.queries(ctx).DeleteItems(ctx, id); err != nil {
		return corepg.TranslateError(err)
	}
	for _, item := range items {
		if err := r.SaveItem(ctx, packageId, item); err != nil {
			return err
		}
	}
	return nil
}

// SaveItem agrega el TC a la selección o guarda su ajuste.
func (r *Repository) SaveItem(ctx context.Context, packageId string, item domain.Item) error {
	id, err := parseID(packageId)
	if err != nil {
		return err
	}
	testCaseID, err := parseID(item.TestCase.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).UpsertItem(ctx, sqlcgen.UpsertItemParams{
		PackageID: id, TestCaseID: testCaseID, Origin: string(item.Origin), Proposed: item.Proposed, Included: item.Included,
		Reason: item.Reason, DecidedBy: nullable(string(item.DecidedBy)), DecidedAt: item.DecidedAt,
	}))
}

// DeleteItem saca un TC de la selección.
func (r *Repository) DeleteItem(ctx context.Context, packageId, testCaseId string) error {
	id, err := parseID(packageId)
	if err != nil {
		return err
	}
	testCaseID, err := parseID(testCaseId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).DeleteItem(ctx, sqlcgen.DeleteItemParams{PackageID: id, TestCaseID: testCaseID}))
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

func text(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// utc normaliza a UTC una fecha opcional leída de la base.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
