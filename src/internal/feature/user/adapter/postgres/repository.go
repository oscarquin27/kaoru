// Package postgres implementa los puertos de user con sqlc.
package postgres

import (
	"context"
	"strings"
	"time"

	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/user/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/user/application/port"
	"kaoru/internal/feature/user/domain"
	shared "kaoru/internal/shared/domain"
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

// List devuelve hasta limit usuarios cuyo nombre o usuario contiene query, por nombre.
func (r *Repository) List(ctx context.Context, query string, limit int) ([]domain.Profile, error) {
	rows, err := sqlcgen.New(r.db.Conn(ctx)).ListUsers(ctx, sqlcgen.ListUsersParams{Query: likeLiteral(query), MaxRows: int32(limit)}) //nolint:gosec // limit lo acota el caso de uso
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	users := make([]domain.Profile, 0, len(rows))
	for _, row := range rows {
		users = append(users, domain.Profile{Id: shared.UserId(row.ID), Username: row.Username, DisplayName: row.DisplayName})
	}
	return users, nil
}

// likeLiteral escapa los comodines de LIKE para buscar el texto tal cual.
func likeLiteral(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
