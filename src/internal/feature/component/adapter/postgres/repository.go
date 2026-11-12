// Package postgres implementa los puertos de component con sqlc.
package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/component/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/component/application/port"
	"kaoru/internal/feature/component/domain"
	shared "kaoru/internal/shared/domain"
)

// uniqueName es la restricción que impide repetir un nombre dentro de un sistema.
const uniqueName = "component_system_id_name_key"

// Repository guarda y lee los COMP.
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
		Prefix: shared.PrefixComponent, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixComponent, year, int(seq)), nil
}

// FindSystem devuelve NOT_FOUND_RESOURCE si el sistema no existe.
func (r *Repository) FindSystem(ctx context.Context, key string) (domain.System, error) {
	row, err := r.queries(ctx).GetSystem(ctx, key)
	if err != nil {
		return domain.System{}, corepg.TranslateError(err)
	}
	return domain.System{Key: row.Key, Name: row.Name, DefaultRoute: row.DefaultRoute}, nil
}

// Insert guarda un COMP nuevo.
func (r *Repository) Insert(ctx context.Context, component domain.Component) error {
	s := component.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	return translate(r.queries(ctx).InsertComponent(ctx, sqlcgen.InsertComponentParams{
		ID: id, Code: s.Code.String(), SystemKey: s.System.Key, Kind: string(s.Kind), Name: s.Name,
		Description: s.Description, RetiredAt: s.RetiredAt, CreatedBy: string(s.CreatedBy),
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Update guarda los cambios de un COMP.
func (r *Repository) Update(ctx context.Context, component domain.Component) error {
	s := component.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateComponent(ctx, sqlcgen.UpdateComponentParams{
		ID: id, Kind: string(s.Kind), Name: s.Name, Description: s.Description,
		RetiredAt: s.RetiredAt, UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return translate(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "El COMP no existe.")
	}
	return nil
}

// Lock bloquea la entidad hasta el fin de la transacción en curso.
func (r *Repository) Lock(ctx context.Context, code shared.Code) error {
	_, err := r.queries(ctx).LockComponent(ctx, code.String())
	return corepg.TranslateError(err)
}

// FindByCode devuelve NOT_FOUND_RESOURCE si el COMP no existe.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (port.ComponentView, error) {
	row, err := r.queries(ctx).GetComponentByCode(ctx, code.String())
	if err != nil {
		return port.ComponentView{}, corepg.TranslateError(err)
	}
	return toView(sqlcgen.ListComponentsRow(row))
}

// FindByCodes devuelve los COMP que existen entre los códigos dados.
func (r *Repository) FindByCodes(ctx context.Context, codes []string) ([]domain.Component, error) {
	rows, err := r.queries(ctx).ListComponentsByCodes(ctx, codes)
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	components := make([]domain.Component, len(rows))
	for i, row := range rows {
		view, err := toView(sqlcgen.ListComponentsRow(row))
		if err != nil {
			return nil, err
		}
		components[i] = view.Component
	}
	return components, nil
}

// List devuelve una página de COMP, por nombre.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]port.ComponentView, int, error) {
	q := r.queries(ctx)
	total, err := q.CountComponents(ctx, sqlcgen.CountComponentsParams{
		SystemKey: nullable(f.SystemKey), Kind: nullable(f.Kind), Query: nullable(f.Query), IncludeRetired: f.IncludeRetired,
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListComponents(ctx, sqlcgen.ListComponentsParams{
		SystemKey: nullable(f.SystemKey), Kind: nullable(f.Kind), Query: nullable(f.Query), IncludeRetired: f.IncludeRetired,
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	views := make([]port.ComponentView, len(rows))
	for i, row := range rows {
		if views[i], err = toView(row); err != nil {
			return nil, 0, err
		}
	}
	return views, int(total), nil
}

func toView(row sqlcgen.ListComponentsRow) (port.ComponentView, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return port.ComponentView{}, axis.Wrap(axis.InternalServerError, "código de COMP no válido en la base", err)
	}
	snapshot := domain.Snapshot{
		Id: row.ID.String(), Code: code,
		System: domain.System{Key: row.SystemKey, Name: row.SystemName, DefaultRoute: row.SystemDefaultRoute},
		Kind:   domain.Kind(row.Kind), Name: row.Name, Description: row.Description,
		// Las fechas se devuelven en UTC, como las entrega el reloj.
		CreatedBy: shared.UserId(row.CreatedBy), CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	if row.RetiredAt != nil {
		retired := row.RetiredAt.UTC()
		snapshot.RetiredAt = &retired
	}
	return port.ComponentView{Component: domain.Rehydrate(snapshot), CreatedByName: row.CreatedByName}, nil
}

// translate convierte el choque con el nombre único en su error de dominio.
func translate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == uniqueName {
		return domain.ErrNameTaken
	}
	return corepg.TranslateError(err)
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id de COMP no válido", err)
	}
	return id, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
