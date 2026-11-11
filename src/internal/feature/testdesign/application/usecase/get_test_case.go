package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/domain"
)

// GetTestCase devuelve un TC por su ID legible. Leer no exige permiso.
type GetTestCase struct {
	d Deps
}

// GetTestCaseInput es la entrada de GetTestCase.
type GetTestCaseInput struct {
	Code string
}

// Execute busca el TC.
func (uc *GetTestCase) Execute(ctx context.Context, in GetTestCaseInput) (domain.TestCaseSnapshot, error) {
	testCase, err := findTestCase(ctx, uc.d.Repository, in.Code)
	return testCase.Snapshot(), err
}
