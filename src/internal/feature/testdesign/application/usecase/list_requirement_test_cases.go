package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/application/port"
)

// maxRequirementTestCases acota cuántos TC de un REQ se leen de una vez.
const maxRequirementTestCases = 10000

// ListRequirementTestCases devuelve todos los TC que validan algún UC de un REQ.
// Lo usan otros features: execution calcula con él la cobertura.
type ListRequirementTestCases struct {
	d Deps
}

// ListRequirementTestCasesInput es la entrada de ListRequirementTestCases.
type ListRequirementTestCasesInput struct {
	RequirementCode string
}

// RequirementTestCase es un TC de un REQ, con lo mínimo para ejecutarlo o contarlo.
type RequirementTestCase struct {
	Id      string
	Code    string
	Retired bool
}

// Execute devuelve NOT_FOUND_RESOURCE si el REQ no existe.
func (uc *ListRequirementTestCases) Execute(ctx context.Context, in ListRequirementTestCasesInput) ([]RequirementTestCase, error) {
	requirement, err := uc.d.Requirements.Find(ctx, in.RequirementCode)
	if err != nil {
		return nil, err
	}
	testCases, _, err := uc.d.Repository.ListTestCases(ctx, port.TestCaseFilter{
		RequirementId: requirement.Id, Page: 1, Size: maxRequirementTestCases,
	})
	if err != nil {
		return nil, err
	}
	out := make([]RequirementTestCase, len(testCases))
	for i, t := range testCases {
		s := t.Snapshot()
		out[i] = RequirementTestCase{Id: s.Id, Code: s.Code.String(), Retired: s.RetiredAt != nil}
	}
	return out, nil
}
