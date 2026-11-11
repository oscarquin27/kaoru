package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
)

// GetInitialState devuelve el estado en el que nace una entidad.
type GetInitialState struct {
	repo port.Repository
}

// GetInitialStateInput es la entrada de GetInitialState.
type GetInitialStateInput struct {
	EntityType string
}

// Execute lee el estado inicial del flujo.
func (uc *GetInitialState) Execute(ctx context.Context, in GetInitialStateInput) (domain.State, error) {
	wf, err := loadWorkflow(ctx, uc.repo, in.EntityType)
	if err != nil {
		return domain.State{}, err
	}
	return wf.Initial()
}
