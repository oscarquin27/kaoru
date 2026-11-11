// Package facts implementa el puerto Facts con los casos de uso de los features que
// dependen de requirement. Como esos features se arman después, bootstrap conecta cada
// fuente una vez construidos todos (reglas §A.7, decisión D36).
package facts

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	defectuc "kaoru/internal/feature/defect/application/usecase"
	executionuc "kaoru/internal/feature/execution/application/usecase"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	riskuc "kaoru/internal/feature/risk/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Facts reúne las fuentes de hechos. Una fuente sin conectar responde «no hay»:
// sin RISK, sin TC y sin defectos, que es lo que bloquea por defecto.
type Facts struct {
	risk     *riskuc.GetRisk
	coverage *executionuc.GetCoverage
	defects  *defectuc.ListOpenDefects
}

var _ port.Facts = (*Facts)(nil)

// New crea el adaptador sin fuentes.
func New() *Facts { return &Facts{} }

// ConnectRisk conecta el feature risk. Se llama una sola vez, al arrancar.
func (f *Facts) ConnectRisk(get *riskuc.GetRisk) { f.risk = get }

// ConnectCoverage conecta el feature execution. Se llama una sola vez, al arrancar.
func (f *Facts) ConnectCoverage(get *executionuc.GetCoverage) { f.coverage = get }

// ConnectDefects conecta el feature defect. Se llama una sola vez, al arrancar.
func (f *Facts) ConnectDefects(list *defectuc.ListOpenDefects) { f.defects = list }

// Risk devuelve nil si el riesgo no está evaluado.
func (f *Facts) Risk(ctx context.Context, code shared.Code) (*domain.RiskFact, error) {
	if f.risk == nil {
		return nil, nil
	}
	out, err := f.risk.Execute(ctx, riskuc.GetRiskInput{RequirementCode: code.String()})
	if axis.IsCode(err, axis.NotFoundResource) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.RiskFact{Level: string(out.Level), Characteristics: len(out.Characteristics)}, nil
}

// Coverage devuelve cuántos TC exige el REQ y cuáles no tienen su último RUN aprobado.
func (f *Facts) Coverage(ctx context.Context, code shared.Code) (domain.CoverageFact, error) {
	if f.coverage == nil {
		return domain.CoverageFact{}, nil
	}
	out, err := f.coverage.Execute(ctx, executionuc.GetCoverageInput{RequirementCode: code.String()})
	if err != nil {
		return domain.CoverageFact{}, err
	}
	return domain.CoverageFact{Required: out.Required, Pending: out.Pending}, nil
}

// OpenDefects devuelve los defectos del REQ que no están cerrados.
func (f *Facts) OpenDefects(ctx context.Context, code shared.Code) ([]domain.DefectFact, error) {
	if f.defects == nil {
		return nil, nil
	}
	open, err := f.defects.Execute(ctx, defectuc.ListOpenDefectsInput{RequirementCode: code.String()})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DefectFact, len(open))
	for i, d := range open {
		out[i] = domain.DefectFact{Code: d.Code, Severity: d.Severity}
	}
	return out, nil
}
