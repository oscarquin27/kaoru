// Package postgres implementa los puertos de savedview con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/savedview/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/savedview/application/port"
	"kaoru/internal/feature/savedview/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda las vistas de cada usuario.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// Insert guarda una vista.
func (r *Repository) Insert(ctx context.Context, v domain.View) error {
	id, err := uuid.Parse(v.Id)
	if err != nil {
		return axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return corepg.TranslateError(r.queries(ctx).InsertView(ctx, sqlcgen.InsertViewParams{
		ID: id, OwnerID: string(v.OwnerId), Screen: v.Screen, Name: v.Name, Query: v.Query, CreatedAt: v.CreatedAt,
	}))
}

// List devuelve las vistas del dueño en una lista, por nombre.
func (r *Repository) List(ctx context.Context, owner shared.UserId, screen string) ([]domain.View, error) {
	rows, err := r.queries(ctx).ListViews(ctx, sqlcgen.ListViewsParams{OwnerID: string(owner), Screen: screen})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	views := make([]domain.View, len(rows))
	for i, row := range rows {
		views[i] = domain.View{Id: row.ID.String(), OwnerId: owner, Screen: screen, Name: row.Name, Query: row.Query, CreatedAt: row.CreatedAt.UTC()}
	}
	return views, nil
}

// Count dice cuántas vistas tiene el dueño en una lista.
func (r *Repository) Count(ctx context.Context, owner shared.UserId, screen string) (int, error) {
	n, err := r.queries(ctx).CountViews(ctx, sqlcgen.CountViewsParams{OwnerID: string(owner), Screen: screen})
	return int(n), corepg.TranslateError(err)
}

// Delete quita una vista del dueño. Un id que no es un UUID tampoco existe.
func (r *Repository) Delete(ctx context.Context, owner shared.UserId, id string) error {
	viewID, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrNotFound
	}
	rows, err := r.queries(ctx).DeleteView(ctx, sqlcgen.DeleteViewParams{ID: viewID, OwnerID: string(owner)})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
