package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/comment/domain"
)

// ListComments devuelve los comentarios de una entidad. Leer no exige permiso.
type ListComments struct {
	d Deps
}

// ListCommentsInput es la entrada de ListComments.
type ListCommentsInput struct {
	EntityType string
	EntityCode string
}

// Execute devuelve una lista vacía si la entidad no existe.
func (uc *ListComments) Execute(ctx context.Context, in ListCommentsInput) ([]domain.Comment, error) {
	entity, err := uc.d.Entities.Resolve(ctx, in.EntityType, in.EntityCode)
	if axis.IsCode(err, axis.NotFoundResource) {
		return []domain.Comment{}, nil
	}
	if err != nil {
		return nil, err
	}
	return uc.d.Repository.List(ctx, entity)
}
