// Package requirement implementa el puerto Requirements con los casos de uso de requirement.
package requirement

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Requirements busca REQ en el feature requirement y los pasa a «En paquete».
type Requirements struct {
	get        *requirementuc.GetRequirement
	transition *requirementuc.TransitionRequirement
}

var _ port.Requirements = (*Requirements)(nil)

// New crea el adaptador.
func New(get *requirementuc.GetRequirement, transition *requirementuc.TransitionRequirement) *Requirements {
	return &Requirements{get: get, transition: transition}
}

// Find conserva el código de error de requirement.
func (r *Requirements) Find(ctx context.Context, code string) (domain.Requirement, error) {
	s, err := r.get.Execute(ctx, requirementuc.GetRequirementInput{Code: code})
	if err != nil {
		return domain.Requirement{}, err
	}
	return domain.Requirement{
		Ref: domain.RequirementRef{
			Id: s.Id, Code: s.Code.String(), Title: s.Title, State: domain.RequirementState{Key: s.State.Key, Name: s.State.Name},
		},
		SystemKey: s.System.Key,
	}, nil
}

// MoveToPackage pasa el REQ a «En paquete» con la transición de su flujo: exige su permiso, evalúa
// sus reglas y la audita requirement. Reutiliza la transacción del context y conserva el código de error.
func (r *Requirements) MoveToPackage(ctx context.Context, actor shared.Actor, code string) error {
	_, err := r.transition.Execute(ctx, requirementuc.TransitionRequirementInput{
		Actor: actor, Code: code, To: domain.RequirementInPackage,
	})
	return err
}
