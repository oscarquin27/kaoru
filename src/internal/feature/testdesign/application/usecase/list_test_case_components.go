package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/domain"
)

// ListTestCaseComponents devuelve los componentes que ejercita un TC. Leer no exige permiso.
type ListTestCaseComponents struct {
	d Deps
}

// ListTestCaseComponentsInput es la entrada de ListTestCaseComponents.
type ListTestCaseComponentsInput struct {
	Code string
}

// Execute lista los componentes por nombre.
func (uc *ListTestCaseComponents) Execute(ctx context.Context, in ListTestCaseComponentsInput) ([]domain.ComponentRef, error) {
	testCase, err := findTestCase(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return nil, err
	}
	return uc.d.Repository.ListTestCaseComponents(ctx, testCase.Snapshot().Id)
}
