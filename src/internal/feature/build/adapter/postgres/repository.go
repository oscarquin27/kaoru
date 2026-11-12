// Package postgres implementa los puertos de build con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/build/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/build/application/port"
	"kaoru/internal/feature/build/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los BUILD.
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
		Prefix: shared.PrefixBuild, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixBuild, year, int(seq)), nil
}

// FindSystem devuelve NOT_FOUND_RESOURCE si el sistema no existe.
func (r *Repository) FindSystem(ctx context.Context, key string) (domain.System, error) {
	row, err := r.queries(ctx).GetSystem(ctx, key)
	if err != nil {
		return domain.System{}, corepg.TranslateError(err)
	}
	return domain.System{Key: row.Key, Name: row.Name, DefaultRoute: row.DefaultRoute}, nil
}

// Insert guarda un BUILD con su manifiesto. No existe una operación para editarlo ni borrarlo.
func (r *Repository) Insert(ctx context.Context, build domain.Build) error {
	s := build.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	q := r.queries(ctx)
	if err := q.InsertBuild(ctx, sqlcgen.InsertBuildParams{
		ID: id, Code: s.Code.String(), SystemKey: s.System.Key, Version: s.Version, Checksum: s.Checksum,
		Migrations: s.Migrations, Dependencies: s.Dependencies, Notes: s.Notes,
		BuiltBy: string(s.BuiltBy.Id), CreatedAt: s.CreatedAt,
	}); err != nil {
		return corepg.TranslateError(err)
	}
	for _, requirement := range s.Requirements {
		requirementID, err := parseID(requirement.Id)
		if err != nil {
			return err
		}
		if err := q.InsertBuildRequirement(ctx, sqlcgen.InsertBuildRequirementParams{BuildID: id, RequirementID: requirementID}); err != nil {
			return corepg.TranslateError(err)
		}
	}
	return nil
}

// FindByCode devuelve NOT_FOUND_RESOURCE si el BUILD no existe.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (domain.Build, error) {
	row, err := r.queries(ctx).GetBuildByCode(ctx, code.String())
	if err != nil {
		return domain.Build{}, corepg.TranslateError(err)
	}
	builds, err := r.withRequirements(ctx, []sqlcgen.ListBuildsRow{sqlcgen.ListBuildsRow(row)})
	if err != nil {
		return domain.Build{}, err
	}
	return builds[0], nil
}

// FindCodeByChecksum devuelve NOT_FOUND_RESOURCE si ningún BUILD tiene ese checksum.
func (r *Repository) FindCodeByChecksum(ctx context.Context, checksum string) (shared.Code, error) {
	raw, err := r.queries(ctx).GetBuildCodeByChecksum(ctx, checksum)
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return parseCode(raw)
}

// List devuelve una página de BUILD, del más reciente al más antiguo.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]domain.Build, int, error) {
	q := r.queries(ctx)
	total, err := q.CountBuilds(ctx, sqlcgen.CountBuildsParams{
		SystemKey: nullable(f.SystemKey), Query: nullable(f.Query), RequirementCode: nullable(f.RequirementCode),
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListBuilds(ctx, sqlcgen.ListBuildsParams{
		SystemKey: nullable(f.SystemKey), Query: nullable(f.Query), RequirementCode: nullable(f.RequirementCode),
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	builds, err := r.withRequirements(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return builds, int(total), nil
}

// withRequirements arma los BUILD de esas filas con los REQ de su manifiesto, leídos en una sola consulta.
func (r *Repository) withRequirements(ctx context.Context, rows []sqlcgen.ListBuildsRow) ([]domain.Build, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	links, err := r.queries(ctx).ListBuildRequirements(ctx, ids)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	requirements := make(map[uuid.UUID][]domain.RequirementRef, len(rows))
	for _, l := range links {
		requirements[l.BuildID] = append(requirements[l.BuildID], domain.RequirementRef{
			Id: l.ID.String(), Code: l.Code, Title: l.Title, SystemKey: l.SystemKey,
		})
	}
	builds := make([]domain.Build, len(rows))
	for i, row := range rows {
		code, err := parseCode(row.Code)
		if err != nil {
			return nil, err
		}
		builds[i] = domain.Rehydrate(domain.Snapshot{
			Id: row.ID.String(), Code: code,
			System:  domain.System{Key: row.SystemKey, Name: row.SystemName, DefaultRoute: row.SystemDefaultRoute},
			Version: row.Version, Checksum: row.Checksum, Requirements: requirements[row.ID],
			Migrations: row.Migrations, Dependencies: row.Dependencies, Notes: row.Notes,
			BuiltBy:   domain.UserRef{Id: shared.UserId(row.BuiltBy), Name: row.BuiltByName},
			CreatedAt: row.CreatedAt.UTC(),
		})
	}
	return builds, nil
}

func parseCode(raw string) (shared.Code, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return shared.Code{}, axis.Wrap(axis.InternalServerError, "código de BUILD no válido en la base", err)
	}
	return code, nil
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
