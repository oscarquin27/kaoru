package usecase

import (
	"context"

	"kaoru/internal/feature/execution/domain"
)

// GetCoverage calcula la cobertura de un REQ: cuántos TC exige y cuáles no tienen
// su último RUN aprobado. Lo usa requirement para sus reglas de bloqueo.
type GetCoverage struct {
	d Deps
}

// GetCoverageInput es la entrada de GetCoverage.
type GetCoverageInput struct {
	RequirementCode string
}

// GetCoverageOutput es la cobertura de un REQ.
type GetCoverageOutput struct {
	// Required son los TC activos que validan algún UC del REQ.
	Required int
	// Pending son los códigos de los TC cuyo último RUN no está aprobado, o que no tienen RUN.
	Pending []string
}

// Execute devuelve NOT_FOUND_RESOURCE si el REQ no existe. Leer no exige permiso.
func (uc *GetCoverage) Execute(ctx context.Context, in GetCoverageInput) (GetCoverageOutput, error) {
	testCases, err := uc.d.TestDesign.RequirementTestCases(ctx, in.RequirementCode)
	if err != nil {
		return GetCoverageOutput{}, err
	}
	ids := make([]string, len(testCases))
	for i, tc := range testCases {
		ids[i] = tc.Id
	}
	latest, err := uc.d.Repository.LatestResults(ctx, ids)
	if err != nil {
		return GetCoverageOutput{}, err
	}
	required, pending := domain.Coverage(testCases, latest)
	return GetCoverageOutput{Required: required, Pending: pending}, nil
}
