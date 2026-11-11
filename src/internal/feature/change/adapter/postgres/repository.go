// Package postgres implementa los puertos de change con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/change/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/change/application/port"
	"kaoru/internal/feature/change/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los CHG.
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
		Prefix: shared.PrefixChange, Year: int32(year), //nolint:gosec // un año cabe en int32
	})
	if err != nil {
		return shared.Code{}, corepg.TranslateError(err)
	}
	return shared.NewCode(shared.PrefixChange, year, int(seq)), nil
}

// FindSystem devuelve NOT_FOUND_RESOURCE si el sistema no existe.
func (r *Repository) FindSystem(ctx context.Context, key string) (domain.System, error) {
	row, err := r.queries(ctx).GetSystem(ctx, key)
	if err != nil {
		return domain.System{}, corepg.TranslateError(err)
	}
	return domain.System{Key: row.Key, Name: row.Name, DefaultRoute: domain.Route(row.DefaultRoute)}, nil
}

// Insert guarda un CHG nuevo.
func (r *Repository) Insert(ctx context.Context, change domain.Change) error {
	s := change.Snapshot()
	id, stateID, err := parseIDs(s)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertChange(ctx, sqlcgen.InsertChangeParams{
		ID: id, Code: s.Code.String(), Title: s.Title, Description: s.Description, SystemKey: s.System.Key,
		Route: string(s.Route), StateID: stateID, WindowStart: s.WindowStart, WindowEnd: s.WindowEnd,
		CreatedBy: string(s.CreatedBy), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Update guarda los cambios de un CHG.
func (r *Repository) Update(ctx context.Context, change domain.Change) error {
	s := change.Snapshot()
	id, stateID, err := parseIDs(s)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateChange(ctx, sqlcgen.UpdateChangeParams{
		ID: id, Title: s.Title, Description: s.Description, Route: string(s.Route), StateID: stateID,
		WindowStart: s.WindowStart, WindowEnd: s.WindowEnd, UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "El CHG no existe.")
	}
	return nil
}

// FindByCode devuelve NOT_FOUND_RESOURCE si el CHG no existe.
func (r *Repository) FindByCode(ctx context.Context, code shared.Code) (port.ChangeView, error) {
	row, err := r.queries(ctx).GetChangeByCode(ctx, code.String())
	if err != nil {
		return port.ChangeView{}, corepg.TranslateError(err)
	}
	return toView(sqlcgen.ListChangesRow(row))
}

// List devuelve una página de CHG.
func (r *Repository) List(ctx context.Context, f port.Filter) ([]port.ChangeView, int, error) {
	q := r.queries(ctx)
	total, err := q.CountChanges(ctx, sqlcgen.CountChangesParams{
		SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey), Query: nullable(f.Query),
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	rows, err := q.ListChanges(ctx, sqlcgen.ListChangesParams{
		SystemKey: nullable(f.SystemKey), StateKey: nullable(f.StateKey), Query: nullable(f.Query),
		PageSize: int32(f.Size), PageOffset: int32((f.Page - 1) * f.Size), //nolint:gosec // acotados por el handler
	})
	if err != nil {
		return nil, 0, corepg.TranslateError(err)
	}
	views := make([]port.ChangeView, len(rows))
	for i, row := range rows {
		if views[i], err = toView(row); err != nil {
			return nil, 0, err
		}
	}
	return views, int(total), nil
}

func toView(row sqlcgen.ListChangesRow) (port.ChangeView, error) {
	code, err := shared.ParseCode(row.Code)
	if err != nil {
		return port.ChangeView{}, axis.Wrap(axis.InternalServerError, "código de CHG no válido en la base", err)
	}
	snapshot := domain.Snapshot{
		Id: row.ID.String(), Code: code, Title: row.Title, Description: row.Description,
		System: domain.System{Key: row.SystemKey, Name: row.SystemName, DefaultRoute: domain.Route(row.SystemDefaultRoute)},
		Route:  domain.Route(row.Route),
		State:  domain.State{Id: row.StateID.String(), Key: row.StateKey, Name: row.StateName},
		// Las fechas se devuelven en UTC, como las entrega el reloj.
		WindowStart: utc(row.WindowStart), WindowEnd: utc(row.WindowEnd),
		CreatedBy: shared.UserId(row.CreatedBy), CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	return port.ChangeView{Change: domain.Rehydrate(snapshot), CreatedByName: row.CreatedByName}, nil
}

func parseIDs(s domain.Snapshot) (id, stateID uuid.UUID, err error) {
	if id, err = uuid.Parse(s.Id); err != nil {
		return id, stateID, axis.Wrap(axis.InternalServerError, "id de CHG no válido", err)
	}
	if stateID, err = uuid.Parse(s.State.Id); err != nil {
		return id, stateID, axis.Wrap(axis.InternalServerError, "id de estado no válido", err)
	}
	return id, stateID, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Lock bloquea la entidad hasta el fin de la transacción en curso.
func (r *Repository) Lock(ctx context.Context, code shared.Code) error {
	_, err := r.queries(ctx).LockChange(ctx, code.String())
	return corepg.TranslateError(err)
}
