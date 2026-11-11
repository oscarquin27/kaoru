package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/domain"
)

// ListDocuments devuelve los documentos enlazados a una entidad. Leer no exige permiso.
type ListDocuments struct {
	d Deps
}

// ListDocumentsInput es la entrada de ListDocuments.
type ListDocumentsInput struct {
	EntityType string
	EntityCode string
}

// Execute devuelve una lista vacía si la entidad no existe.
func (uc *ListDocuments) Execute(ctx context.Context, in ListDocumentsInput) ([]domain.Snapshot, error) {
	entity, err := uc.d.Entities.Resolve(ctx, in.EntityType, in.EntityCode)
	if axis.IsCode(err, axis.NotFoundResource) {
		return []domain.Snapshot{}, nil
	}
	if err != nil {
		return nil, err
	}
	documents, err := uc.d.Repository.ListByEntity(ctx, entity)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Snapshot, len(documents))
	for i, d := range documents {
		out[i] = d.Snapshot()
	}
	return out, nil
}
