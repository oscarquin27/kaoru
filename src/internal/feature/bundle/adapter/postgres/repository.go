// Package postgres implementa los puertos de bundle con sqlc.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/bundle/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los PKG con sus BUILD y sus REQ.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// NextCode reserva el siguiente ID legible del año.
func (r *Repository) NextCode(ctx context.Context, year int) (shared.Code, error) {
	seq, err := r.queries(ctx).NextCode(ctx, sqlcgen.NextCodeParams{
		Prefix: shared.PrefixPackage, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixPackage, year, int(seq)), nil
}

// FindSystem devuelve NOT_FOUND_RESOURCE si el sistema no existe.
func (r *Repository) FindSystem(ctx context.Context, key string) (domain.System, error) {
	row, err := r.queries(ctx).GetSystem(ctx, key)
	if err != nil {
		return domain.System{}, corepg.TranslateError(err)
	}
	return domain.System{Key: row.Key, Name: row.Name, DefaultRoute: row.DefaultRoute}, nil
}

// Insert guarda un PKG nuevo, todavía sin contenido.
func (r *Repository) Insert(ctx context.Context, pkg domain.Package) error {
	s := pkg.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	stateID, err := parseID(s.State.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertPackage(ctx, sqlcgen.InsertPackageParams{
		ID: id, Code: s.Code.String(), Title: s.Title, Description: s.Description, SystemKey: s.System.Key,
		StateID: stateID, CreatedBy: string(s.CreatedBy.Id), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Update guarda el título, la descripción, el estado y la fecha de cambio de un PKG.
func (r *Repository) Update(ctx context.Context, pkg domain.Package) error {
	s := pkg.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	stateID, err := parseID(s.State.Id)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdatePackage(ctx, sqlcgen.UpdatePackageParams{
		ID: id, Title: s.Title, Description: s.Description, StateID: stateID, UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "El PKG no existe.")
	}
	return nil
}

// Lock bloquea la entidad hasta el fin de la transacción en curso.
func (r *Repository) Lock(ctx context.Context, code shared.Code) error {
	_, err := r.queries(ctx).LockPackage(ctx, code.String())
	return corepg.TranslateError(err)
}

// FindByCode devuelve NOT_FOUND_RESOURCE si el PKG no existe.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (domain.Package, error) {
	row, err := r.queries(ctx).GetPackageByCode(ctx, code.String())
	if err != nil {
		return domain.Package{}, corepg.TranslateError(err)
	}
	packages, err := r.withContent(ctx, []sqlcgen.ListPackagesRow{sqlcgen.ListPackagesRow(row)})
	if err != nil {
		return domain.Package{}, err
	}
	return packages[0], nil
}

// List devuelve una página de PKG con su contenido.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]domain.Package, int, error) {
	q := r.queries(ctx)
	total, err := q.CountPackages(ctx, sqlcgen.CountPackagesParams{
		SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey), Query: nullable(f.Query),
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListPackages(ctx, sqlcgen.ListPackagesParams{
		SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey), Query: nullable(f.Query),
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	packages, err := r.withContent(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return packages, int(total), nil
}

// withContent arma los PKG de esas filas con sus BUILD y sus REQ, leídos de una vez para todos.
func (r *Repository) withContent(ctx context.Context, rows []sqlcgen.ListPackagesRow) ([]domain.Package, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	q := r.queries(ctx)
	buildRows, err := q.ListPackageBuilds(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	builds := make(map[uuid.UUID][]domain.BuildRef)
	for _, b := range buildRows {
		builds[b.PackageID] = append(builds[b.PackageID], domain.BuildRef{
			Id: b.ID.String(), Code: b.Code, Version: b.Version, Checksum: b.Checksum,
		})
	}
	requirementRows, err := q.ListPackageRequirements(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	requirements := make(map[uuid.UUID][]domain.RequirementRef)
	for _, req := range requirementRows {
		requirements[req.PackageID] = append(requirements[req.PackageID], domain.RequirementRef{
			Id: req.ID.String(), Code: req.Code, Title: req.Title,
			State: domain.RequirementState{Key: req.StateKey, Name: req.StateName},
		})
	}
	packages := make([]domain.Package, len(rows))
	for i, row := range rows {
		code, err := shared.ParseCode(row.Code)
		if err != nil {
			return nil, axis.Wrap(axis.InternalServerError, "código de PKG no válido en la base", err)
		}
		packages[i] = domain.Rehydrate(domain.Snapshot{
			Id: row.ID.String(), Code: code, Title: row.Title, Description: row.Description,
			System: domain.System{Key: row.SystemKey, Name: row.SystemName, DefaultRoute: row.SystemDefaultRoute},
			State:  domain.State{Id: row.StateID.String(), Key: row.StateKey, Name: row.StateName, IsInitial: row.StateIsInitial},
			Builds: builds[row.ID], Requirements: requirements[row.ID],
			CreatedBy: domain.UserRef{Id: shared.UserId(row.CreatedBy), Name: row.CreatedByName},
			CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
		})
	}
	return packages, nil
}

// AddBuild agrega un BUILD al contenido del PKG.
func (r *Repository) AddBuild(ctx context.Context, packageId, buildId string, by shared.UserId, at time.Time) error {
	id, buildID, err := parseIDs(packageId, buildId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertPackageBuild(ctx, sqlcgen.InsertPackageBuildParams{
		PackageID: id, BuildID: buildID, AddedBy: string(by), AddedAt: at,
	}))
}

// RemoveBuild quita un BUILD del contenido del PKG.
func (r *Repository) RemoveBuild(ctx context.Context, packageId, buildId string) error {
	id, buildID, err := parseIDs(packageId, buildId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).DeletePackageBuild(ctx, sqlcgen.DeletePackageBuildParams{PackageID: id, BuildID: buildID}))
}

// AddRequirement agrega un REQ al contenido del PKG.
func (r *Repository) AddRequirement(ctx context.Context, packageId, requirementId string, by shared.UserId, at time.Time) error {
	id, requirementID, err := parseIDs(packageId, requirementId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertPackageRequirement(ctx, sqlcgen.InsertPackageRequirementParams{
		PackageID: id, RequirementID: requirementID, AddedBy: string(by), AddedAt: at,
	}))
}

// RemoveRequirement quita un REQ del contenido del PKG.
func (r *Repository) RemoveRequirement(ctx context.Context, packageId, requirementId string) error {
	id, requirementID, err := parseIDs(packageId, requirementId)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).DeletePackageRequirement(ctx, sqlcgen.DeletePackageRequirementParams{
		PackageID: id, RequirementID: requirementID,
	}))
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}

func parseIDs(first, second string) (a, b uuid.UUID, err error) {
	if a, err = parseID(first); err != nil {
		return a, b, err
	}
	b, err = parseID(second)
	return a, b, err
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
