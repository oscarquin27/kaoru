// Package postgres implementa los puertos de environment con sqlc.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/environment/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/environment/application/port"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee ENV y DEP.
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

// InsertEnvironment guarda un ENV nuevo.
func (r *Repository) InsertEnvironment(ctx context.Context, environment domain.Environment) error {
	s := environment.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertEnvironment(ctx, sqlcgen.InsertEnvironmentParams{
		ID: id, Code: s.Code.String(), SystemKey: s.System.Key, Name: s.Name, Kind: string(s.Kind), Os: string(s.OS),
		Notes: s.Notes, RetiredAt: s.RetiredAt, CreatedBy: string(s.CreatedBy.Id), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// UpdateEnvironment guarda los cambios de un ENV.
func (r *Repository) UpdateEnvironment(ctx context.Context, environment domain.Environment) error {
	s := environment.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateEnvironment(ctx, sqlcgen.UpdateEnvironmentParams{
		ID: id, Name: s.Name, Kind: string(s.Kind), Os: string(s.OS), Notes: s.Notes, RetiredAt: s.RetiredAt, UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "El ambiente no existe.")
	}
	return nil
}

// LockEnvironment bloquea el ENV hasta el fin de la transacción en curso.
func (r *Repository) LockEnvironment(ctx context.Context, code shared.Code) error {
	_, err := r.queries(ctx).LockEnvironment(ctx, code.String())
	return corepg.TranslateError(err)
}

// FindEnvironment devuelve NOT_FOUND_RESOURCE si el ENV no existe.
func (r *Repository) FindEnvironment(ctx context.Context, code shared.Code) (port.EnvironmentView, error) {
	row, err := r.queries(ctx).GetEnvironmentByCode(ctx, code.String())
	if err != nil {
		return port.EnvironmentView{}, corepg.TranslateError(err)
	}
	return toEnvironmentView(sqlcgen.ListEnvironmentsRow(row))
}

// ListEnvironments devuelve una página de ENV, por sistema y nombre.
func (r *Repository) ListEnvironments(ctx context.Context, f port.EnvironmentFilter) ([]port.EnvironmentView, int, error) {
	q := r.queries(ctx)
	total, err := q.CountEnvironments(ctx, sqlcgen.CountEnvironmentsParams{
		SystemKey: nullable(f.SystemKey), Kind: nullable(f.Kind), IncludeRetired: f.IncludeRetired,
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListEnvironments(ctx, sqlcgen.ListEnvironmentsParams{
		SystemKey: nullable(f.SystemKey), Kind: nullable(f.Kind), IncludeRetired: f.IncludeRetired,
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	views := make([]port.EnvironmentView, len(rows))
	for i, row := range rows {
		if views[i], err = toEnvironmentView(row); err != nil {
			return nil, 0, err
		}
	}
	return views, int(total), nil
}

// FindPackage devuelve NOT_FOUND_RESOURCE si el PKG no existe.
func (r *Repository) FindPackage(ctx context.Context, code string) (domain.PackageRef, error) {
	row, err := r.queries(ctx).GetPackageRef(ctx, code)
	if err != nil {
		return domain.PackageRef{}, corepg.TranslateError(err)
	}
	return domain.PackageRef{Id: row.ID.String(), Code: row.Code, Title: row.Title, SystemKey: row.SystemKey}, nil
}

// InsertDeployment guarda un DEP. No existe una operación para editarlo ni borrarlo.
func (r *Repository) InsertDeployment(ctx context.Context, deployment domain.Deployment) error {
	s := deployment.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	environmentID, err := parseID(s.Environment.Id)
	if err != nil {
		return err
	}
	var buildID, packageID uuid.NullUUID
	if s.Build != nil {
		if buildID.UUID, err = parseID(s.Build.Id); err != nil {
			return err
		}
		buildID.Valid = true
	}
	if s.Package != nil {
		if packageID.UUID, err = parseID(s.Package.Id); err != nil {
			return err
		}
		packageID.Valid = true
	}
	return corepg.TranslateError(r.queries(ctx).InsertDeployment(ctx, sqlcgen.InsertDeploymentParams{
		ID: id, Code: s.Code.String(), EnvironmentID: environmentID, BuildID: buildID, PackageID: packageID,
		Result: string(s.Result), Notes: s.Notes, ExecutedBy: string(s.ExecutedBy.Id),
		ExecutedAt: s.ExecutedAt, CreatedAt: s.CreatedAt,
	}))
}

// FindDeployment devuelve NOT_FOUND_RESOURCE si el DEP no existe.
func (r *Repository) FindDeployment(ctx context.Context, code shared.Code) (domain.Deployment, error) {
	row, err := r.queries(ctx).GetDeploymentByCode(ctx, code.String())
	if err != nil {
		return domain.Deployment{}, corepg.TranslateError(err)
	}
	return toDeployment(sqlcgen.ListDeploymentsRow(row))
}

// ListDeployments devuelve una página de DEP, del más reciente al más antiguo.
func (r *Repository) ListDeployments(ctx context.Context, f port.DeploymentFilter) ([]domain.Deployment, int, error) {
	q := r.queries(ctx)
	total, err := q.CountDeployments(ctx, sqlcgen.CountDeploymentsParams{
		EnvironmentCode: nullable(f.EnvironmentCode), BuildCode: nullable(f.BuildCode), PackageCode: nullable(f.PackageCode),
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListDeployments(ctx, sqlcgen.ListDeploymentsParams{
		EnvironmentCode: nullable(f.EnvironmentCode), BuildCode: nullable(f.BuildCode), PackageCode: nullable(f.PackageCode),
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	deployments := make([]domain.Deployment, len(rows))
	for i, row := range rows {
		if deployments[i], err = toDeployment(row); err != nil {
			return nil, 0, err
		}
	}
	return deployments, int(total), nil
}

// HasProductionDeployment dice si ese PKG tiene un despliegue exitoso en un ambiente de producción.
func (r *Repository) HasProductionDeployment(ctx context.Context, packageCode string) (bool, error) {
	exists, err := r.queries(ctx).HasProductionDeployment(ctx, packageCode)
	return exists, corepg.TranslateError(err)
}

func toEnvironmentView(row sqlcgen.ListEnvironmentsRow) (port.EnvironmentView, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return port.EnvironmentView{}, axis.Wrap(axis.InternalServerError, "código de ENV no válido en la base", err)
	}
	view := port.EnvironmentView{Environment: domain.Rehydrate(domain.Snapshot{
		Id: row.ID.String(), Code: code,
		System: domain.System{Key: row.SystemKey, Name: row.SystemName, DefaultRoute: row.SystemDefaultRoute},
		Name:   row.Name, Kind: domain.Kind(row.Kind), OS: domain.OS(row.Os), Notes: row.Notes, RetiredAt: utc(row.RetiredAt),
		CreatedBy: domain.UserRef{Id: shared.UserId(row.CreatedBy), Name: row.CreatedByName},
		CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	})}
	if row.InstalledDeployment != nil && row.InstalledAt != nil {
		view.Installed = &domain.Installed{
			Deployment: *row.InstalledDeployment, DeployedAt: row.InstalledAt.UTC(),
			Build:   buildRef(row.InstalledBuildID, row.InstalledBuildCode, row.InstalledBuildVersion, row.InstalledBuildChecksum),
			Package: packageRef(row.InstalledPackageID, row.InstalledPackageCode, row.InstalledPackageTitle),
		}
	}
	return view, nil
}

func toDeployment(row sqlcgen.ListDeploymentsRow) (domain.Deployment, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return domain.Deployment{}, axis.Wrap(axis.InternalServerError, "código de DEP no válido en la base", err)
	}
	return domain.RehydrateDeployment(domain.DeploymentSnapshot{
		Id: row.ID.String(), Code: code,
		Environment: domain.EnvironmentRef{
			Id: row.EnvironmentID.String(), Code: row.EnvironmentCode, Name: row.EnvironmentName, Kind: domain.Kind(row.EnvironmentKind),
		},
		Build:   buildRef(row.BuildID, row.BuildCode, row.BuildVersion, row.BuildChecksum),
		Package: packageRef(row.PackageID, row.PackageCode, row.PackageTitle),
		Result:  domain.Result(row.Result), Notes: row.Notes,
		ExecutedBy: domain.UserRef{Id: shared.UserId(row.ExecutedBy), Name: row.ExecutedByName},
		ExecutedAt: row.ExecutedAt.UTC(), CreatedAt: row.CreatedAt.UTC(),
	}), nil
}

// buildRef devuelve nil si el despliegue no fue de un BUILD.
func buildRef(id uuid.NullUUID, code, version, checksum *string) *domain.BuildRef {
	if !id.Valid || code == nil || version == nil || checksum == nil {
		return nil
	}
	return &domain.BuildRef{Id: id.UUID.String(), Code: *code, Version: *version, Checksum: *checksum}
}

// packageRef devuelve nil si el despliegue no fue de un PKG.
func packageRef(id uuid.NullUUID, code, title *string) *domain.PackageRef {
	if !id.Valid || code == nil || title == nil {
		return nil
	}
	return &domain.PackageRef{Id: id.UUID.String(), Code: *code, Title: *title}
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

// utc normaliza a UTC una fecha opcional leída de la base.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
