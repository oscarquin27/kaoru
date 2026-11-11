package usecase

import (
	"context"

	"kaoru/internal/feature/requirement/domain"
)

// GetRequirement devuelve un REQ por su ID legible. Leer no exige permiso.
type GetRequirement struct {
	d Deps
}

// GetRequirementInput es la entrada de GetRequirement.
type GetRequirementInput struct {
	Code string
}

// Execute busca el REQ.
func (uc *GetRequirement) Execute(ctx context.Context, in GetRequirementInput) (domain.Snapshot, error) {
	requirement, err := findRequirement(ctx, uc.d.Repository, in.Code)
	return requirement.Snapshot(), err
}
