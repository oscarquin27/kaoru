// Package requirement implementa el puerto Requirements con los casos de uso de requirement.
package requirement

import (
	"context"

	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/risk/application/port"
)

// Requirements busca REQ en el feature requirement.
type Requirements struct {
	get *requirementuc.GetRequirement
}

var _ port.Requirements = (*Requirements)(nil)

// New crea el adaptador.
func New(get *requirementuc.GetRequirement) *Requirements { return &Requirements{get: get} }

// Find conserva el código de error de requirement.
func (r *Requirements) Find(ctx context.Context, code string) (port.RequirementRef, error) {
	s, err := r.get.Execute(ctx, requirementuc.GetRequirementInput{Code: code})
	if err != nil {
		return port.RequirementRef{}, err
	}
	return port.RequirementRef{Id: s.Id, Code: s.Code.String()}, nil
}
