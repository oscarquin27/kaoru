package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
)

// GetWorkflow devuelve el flujo de un tipo de entidad, para pintar la línea de vida.
type GetWorkflow struct {
	repo port.Repository
}

// GetWorkflowInput es la entrada de GetWorkflow.
type GetWorkflowInput struct {
	EntityType string
}

// Execute lee el flujo.
func (uc *GetWorkflow) Execute(ctx context.Context, in GetWorkflowInput) (domain.Workflow, error) {
	return loadWorkflow(ctx, uc.repo, in.EntityType)
}
