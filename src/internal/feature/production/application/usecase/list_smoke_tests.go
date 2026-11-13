package usecase

import (
	"context"

	"kaoru/internal/feature/production/domain"
)

// ListSmokeTests devuelve una página de SMOKE, del más reciente al más antiguo.
type ListSmokeTests struct {
	d Deps
}

// ListSmokeTestsOutput es una página de SMOKE.
type ListSmokeTestsOutput struct {
	SmokeTests []domain.SmokeTestSnapshot
	Total      int
}

// Execute lista los SMOKE. Leer no exige permiso.
func (uc *ListSmokeTests) Execute(ctx context.Context, in ListInput) (ListSmokeTestsOutput, error) {
	smokeTests, total, err := uc.d.Repository.ListSmokeTests(ctx, in.filter())
	if err != nil {
		return ListSmokeTestsOutput{}, err
	}
	out := ListSmokeTestsOutput{SmokeTests: make([]domain.SmokeTestSnapshot, len(smokeTests)), Total: total}
	for i, t := range smokeTests {
		out.SmokeTests[i] = t.Snapshot()
	}
	return out, nil
}
