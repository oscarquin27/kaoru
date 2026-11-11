package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/domain"
)

// ListFunctionalCases devuelve los UC de un REQ. Leer no exige permiso.
type ListFunctionalCases struct {
	d Deps
}

// ListFunctionalCasesInput es la entrada de ListFunctionalCases.
type ListFunctionalCasesInput struct {
	RequirementCode string
}

// Execute lista los UC en orden de código.
func (uc *ListFunctionalCases) Execute(ctx context.Context, in ListFunctionalCasesInput) ([]domain.FunctionalCaseSnapshot, error) {
	requirement, err := uc.d.Requirements.Find(ctx, in.RequirementCode)
	if err != nil {
		return nil, err
	}
	functionalCases, err := uc.d.Repository.ListFunctionalCases(ctx, requirement.Id)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FunctionalCaseSnapshot, len(functionalCases))
	for i, f := range functionalCases {
		out[i] = f.Snapshot()
	}
	return out, nil
}
