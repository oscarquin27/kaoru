// Package requirement implementa el puerto Requirements con los casos de uso de requirement.
package requirement

import (
	"context"

	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
)

// Requirements busca REQ en el feature requirement.
type Requirements struct {
	get *requirementuc.GetRequirement
}

var _ port.Requirements = (*Requirements)(nil)

// New crea el adaptador.
func New(get *requirementuc.GetRequirement) *Requirements { return &Requirements{get: get} }

// Find conserva el código de error de requirement.
func (r *Requirements) Find(ctx context.Context, code string) (domain.RequirementRef, error) {
	s, err := r.get.Execute(ctx, requirementuc.GetRequirementInput{Code: code})
	if err != nil {
		return domain.RequirementRef{}, err
	}
	return domain.RequirementRef{Id: s.Id, Code: s.Code.String(), SystemKey: s.System.Key}, nil
}
