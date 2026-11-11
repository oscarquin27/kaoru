// Package testdesign implementa el puerto TestDesign con los casos de uso de testdesign.
package testdesign

import (
	"context"

	"kaoru/internal/feature/defect/application/port"
	testdesignuc "kaoru/internal/feature/testdesign/application/usecase"
)

// TestDesign pregunta a testdesign qué TC validan un REQ.
type TestDesign struct {
	list *testdesignuc.ListRequirementTestCases
}

var _ port.TestDesign = (*TestDesign)(nil)

// New crea el adaptador.
func New(list *testdesignuc.ListRequirementTestCases) *TestDesign { return &TestDesign{list: list} }

// RequirementTestCases conserva el código de error de testdesign.
func (t *TestDesign) RequirementTestCases(ctx context.Context, requirementCode string) ([]string, error) {
	testCases, err := t.list.Execute(ctx, testdesignuc.ListRequirementTestCasesInput{RequirementCode: requirementCode})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(testCases))
	for i, tc := range testCases {
		ids[i] = tc.Id
	}
	return ids, nil
}
