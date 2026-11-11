package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/domain"
)

// GetFunctionalCase devuelve un UC por su ID legible. Leer no exige permiso.
type GetFunctionalCase struct {
	d Deps
}

// GetFunctionalCaseInput es la entrada de GetFunctionalCase.
type GetFunctionalCaseInput struct {
	Code string
}

// Execute busca el UC.
func (uc *GetFunctionalCase) Execute(ctx context.Context, in GetFunctionalCaseInput) (domain.FunctionalCaseSnapshot, error) {
	functionalCase, err := findFunctionalCase(ctx, uc.d.Repository, in.Code)
	return functionalCase.Snapshot(), err
}
