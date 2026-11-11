package usecase

import (
	"context"

	"kaoru/internal/feature/traceability/domain"
)

// ListSystems devuelve los sistemas bajo prueba.
type ListSystems struct {
	d Deps
}

// Execute lista los sistemas en orden de nombre.
func (uc *ListSystems) Execute(ctx context.Context) ([]domain.System, error) {
	return uc.d.Repository.Systems(ctx)
}
