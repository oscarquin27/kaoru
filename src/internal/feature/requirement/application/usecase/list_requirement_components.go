package usecase

import (
	"context"

	"kaoru/internal/feature/requirement/domain"
)

// ListRequirementComponents devuelve los componentes que toca un REQ. Leer no exige permiso.
type ListRequirementComponents struct {
	d Deps
}

// ListRequirementComponentsInput es la entrada de ListRequirementComponents.
type ListRequirementComponentsInput struct {
	Code string
}

// Execute lista los componentes por nombre.
func (uc *ListRequirementComponents) Execute(ctx context.Context, in ListRequirementComponentsInput) ([]domain.ComponentRef, error) {
	requirement, err := findRequirement(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return nil, err
	}
	return uc.d.Repository.ListComponents(ctx, requirement.Snapshot().Id)
}
