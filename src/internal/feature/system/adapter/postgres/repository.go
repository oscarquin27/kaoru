// Package postgres implementa los puertos de system con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/system/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/system/application/port"
	"kaoru/internal/feature/system/domain"
)

// Repository guarda y lee los sistemas.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

func (r *Repository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.db.Conn(ctx))
}

// Insert guarda un sistema nuevo; DUPLICATE_RESOURCE si la clave ya existe.
func (r *Repository) Insert(ctx context.Context, system domain.System) error {
	s := system.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(r.queries(ctx).InsertSystem(ctx, sqlcgen.InsertSystemParams{
		ID: id, Key: s.Key, Name: s.Name, DefaultRoute: string(s.DefaultRoute), CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}))
}

// Update guarda los cambios de un sistema.
func (r *Repository) Update(ctx context.Context, system domain.System) error {
	s := system.Snapshot()
	id, err := parseID(s.Id)
	if err != nil {
		return err
	}
	rows, err := r.queries(ctx).UpdateSystem(ctx, sqlcgen.UpdateSystemParams{
		ID: id, Name: s.Name, DefaultRoute: string(s.DefaultRoute), UpdatedAt: s.UpdatedAt,
	})
	if err != nil {
		return corepg.TranslateError(err)
	}
	if rows == 0 {
		return axis.New(axis.NotFoundResource, "El sistema no existe.")
	}
	return nil
}

// Lock bloquea el sistema y lo lee; NOT_FOUND_RESOURCE si no existe.
func (r *Repository) Lock(ctx context.Context, key string) (domain.System, error) {
	row, err := r.queries(ctx).LockSystem(ctx, key)
	if err != nil {
		return domain.System{}, corepg.TranslateError(err)
	}
	return domain.Rehydrate(domain.Snapshot{
		Id: row.ID.String(), Key: row.Key, Name: row.Name, DefaultRoute: domain.Route(row.DefaultRoute),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}), nil
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}
