package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
)

// ListTestCases devuelve una página de TC en orden de código. Leer no exige permiso.
type ListTestCases struct {
	d Deps
}

// ListTestCasesInput es la entrada de ListTestCases. Un filtro vacío o nil no filtra.
type ListTestCasesInput struct {
	SystemKey       string
	Critical        *bool
	Query           string
	RequirementCode string
	Page            int
	Size            int
}

// ListTestCasesOutput es una página de TC.
type ListTestCasesOutput struct {
	TestCases []domain.TestCaseSnapshot
	Total     int
}

// Execute lista los TC; con RequirementCode, solo los que validan algún UC de ese REQ.
func (uc *ListTestCases) Execute(ctx context.Context, in ListTestCasesInput) (ListTestCasesOutput, error) {
	filter := port.TestCaseFilter{SystemKey: in.SystemKey, Critical: in.Critical, Query: in.Query, Page: in.Page, Size: in.Size}
	if in.RequirementCode != "" {
		requirement, err := uc.d.Requirements.Find(ctx, in.RequirementCode)
		if err != nil {
			return ListTestCasesOutput{}, err
		}
		filter.RequirementId = requirement.Id
	}
	testCases, total, err := uc.d.Repository.ListTestCases(ctx, filter)
	if err != nil {
		return ListTestCasesOutput{}, err
	}
	out := ListTestCasesOutput{TestCases: make([]domain.TestCaseSnapshot, len(testCases)), Total: total}
	for i, t := range testCases {
		out.TestCases[i] = t.Snapshot()
	}
	return out, nil
}
