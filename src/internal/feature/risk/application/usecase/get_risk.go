package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/risk/domain"
)

// GetRisk devuelve el RISK vigente de un REQ. Leer no exige permiso.
type GetRisk struct {
	d Deps
}

// GetRiskInput es la entrada de GetRisk.
type GetRiskInput struct {
	RequirementCode string
}

// Execute devuelve NOT_FOUND_RESOURCE si el REQ no existe o si su riesgo no está evaluado.
func (uc *GetRisk) Execute(ctx context.Context, in GetRiskInput) (domain.Snapshot, error) {
	requirement, err := uc.d.Requirements.Find(ctx, in.RequirementCode)
	if err != nil {
		return domain.Snapshot{}, err
	}
	assessment, err := uc.d.Repository.FindByRequirement(ctx, requirement.Id)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Snapshot{}, ErrRiskNotFound
	}
	return assessment.Snapshot(), err
}
