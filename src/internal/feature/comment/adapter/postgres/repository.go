// Package postgres implementa los puertos de comment con sqlc.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/comment/adapter/postgres/sqlcgen"
	"kaoru/internal/feature/comment/application/port"
	"kaoru/internal/feature/comment/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee comentarios.
type Repository struct {
	db *corepg.Database
}

var _ port.Repository = (*Repository)(nil)

// NewRepository crea el repositorio.
func NewRepository(db *corepg.Database) *Repository { return &Repository{db: db} }

// Insert guarda un comentario.
func (r *Repository) Insert(ctx context.Context, c domain.Comment) error {
	id, err := parseID(c.Id)
	if err != nil {
		return err
	}
	entityID, err := parseID(c.Entity.Id)
	if err != nil {
		return err
	}
	return corepg.TranslateError(sqlcgen.New(r.db.Conn(ctx)).InsertComment(ctx, sqlcgen.InsertCommentParams{
		ID: id, EntityType: c.Entity.Type, EntityID: entityID, Body: c.Body, AuthorID: string(c.AuthorId), CreatedAt: c.CreatedAt,
	}))
}

// List devuelve los comentarios de una entidad, del más antiguo al más reciente.
func (r *Repository) List(ctx context.Context, entity domain.EntityRef) ([]domain.Comment, error) {
	entityID, err := parseID(entity.Id)
	if err != nil {
		return nil, err
	}
	rows, err := sqlcgen.New(r.db.Conn(ctx)).ListComments(ctx, sqlcgen.ListCommentsParams{EntityType: entity.Type, EntityID: entityID})
	if err != nil {
		return nil, corepg.TranslateError(err)
	}
	comments := make([]domain.Comment, len(rows))
	for i, row := range rows {
		comments[i] = domain.Comment{
			Id: row.ID.String(), Entity: entity, Body: row.Body, AuthorId: shared.UserId(row.AuthorID),
			AuthorName: row.AuthorName, CreatedAt: row.CreatedAt.UTC(),
		}
	}
	return comments, nil
}

// FindUserName devuelve el nombre de un usuario.
func (r *Repository) FindUserName(ctx context.Context, id shared.UserId) (string, error) {
	name, err := sqlcgen.New(r.db.Conn(ctx)).GetUserName(ctx, string(id))
	return name, corepg.TranslateError(err)
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return id, axis.Wrap(axis.InternalServerError, "id no válido", err)
	}
	return id, nil
}
