// Package execution implementa el puerto Coverage con los casos de uso de execution.
package execution

import (
	"context"

	executionuc "kaoru/internal/feature/execution/application/usecase"
	"kaoru/internal/feature/traceability/application/port"
	"kaoru/internal/feature/traceability/domain"
)

// Coverage pregunta a execution la cobertura de un REQ: la regla vive allí, no se repite aquí.
type Coverage struct {
	get *executionuc.GetCoverage
}

var _ port.Coverage = (*Coverage)(nil)

// New crea el adaptador.
func New(get *executionuc.GetCoverage) *Coverage { return &Coverage{get: get} }

// Of conserva el código de error de execution.
func (c *Coverage) Of(ctx context.Context, requirementCode string) (domain.Coverage, error) {
	out, err := c.get.Execute(ctx, executionuc.GetCoverageInput{RequirementCode: requirementCode})
	if err != nil {
		return domain.Coverage{}, err
	}
	return domain.Coverage{Required: out.Required, Passed: out.Required - len(out.Pending)}, nil
}
