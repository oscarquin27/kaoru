package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
)

// ListRuns devuelve una página de RUN, del más reciente al más antiguo. Leer no exige permiso.
type ListRuns struct {
	d Deps
}

// ListRunsInput es la entrada de ListRuns. Un filtro vacío no filtra.
type ListRunsInput struct {
	TestCaseCode    string
	RequirementCode string
	BuildVersion    string
	Page            int
	Size            int
}

// ListRunsOutput es una página de RUN.
type ListRunsOutput struct {
	Runs  []domain.RunSnapshot
	Total int
}

// Execute lista los RUN. Filtrar por un TC o un REQ que no existe devuelve una lista vacía.
func (uc *ListRuns) Execute(ctx context.Context, in ListRunsInput) (ListRunsOutput, error) {
	filter := port.RunFilter{BuildVersion: in.BuildVersion, Page: in.Page, Size: in.Size}
	if in.RequirementCode != "" {
		filter.FilterTestCases = true
		testCases, err := uc.d.TestDesign.RequirementTestCases(ctx, in.RequirementCode)
		if err != nil && !axis.IsCode(err, axis.NotFoundResource) {
			return ListRunsOutput{}, err
		}
		for _, tc := range testCases {
			filter.TestCaseIds = append(filter.TestCaseIds, tc.Id)
		}
	}
	if in.TestCaseCode != "" {
		testCase, err := uc.d.Repository.FindTestCase(ctx, in.TestCaseCode)
		if err != nil && !axis.IsCode(err, axis.NotFoundResource) {
			return ListRunsOutput{}, err
		}
		// Con los dos filtros queda el TC solo si además pertenece al REQ.
		keep := err == nil && (!filter.FilterTestCases || contains(filter.TestCaseIds, testCase.Id))
		filter.FilterTestCases, filter.TestCaseIds = true, nil
		if keep {
			filter.TestCaseIds = []string{testCase.Id}
		}
	}
	runs, total, err := uc.d.Repository.ListRuns(ctx, filter)
	if err != nil {
		return ListRunsOutput{}, err
	}
	out := ListRunsOutput{Runs: make([]domain.RunSnapshot, len(runs)), Total: total}
	for i, r := range runs {
		out.Runs[i] = r.Snapshot()
	}
	return out, nil
}

func contains(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}
