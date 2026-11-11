// Package postgres implementa los puertos de user con sqlc.
package postgres

import (
	"context"
	"time"

	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/user/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/user/application/port"
	"kaoru/internal/feature/user/domain"
)

// Repository guarda el perfil de los usuarios.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

// Upsert crea el perfil o lo actualiza si cambió.
func (r *Repository) Upsert(ctx context.Context, p domain.Profile, now time.Time) error {
	var email *string
	if p.Email != "" {
		email = &p.Email
	}
	return corepg.TranslateError(sqlcgen.New(r.db.Conn(ctx)).UpsertUser(ctx, sqlcgen.UpsertUserParams{
		ID: string(p.Id), Username: p.Username, DisplayName: p.DisplayName, Email: email, CreatedAt: now,
	}))
}
