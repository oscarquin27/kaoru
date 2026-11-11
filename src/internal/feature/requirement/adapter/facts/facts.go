// Package facts implementa el puerto Facts con los casos de uso de los features que
// dependen de requirement. Como esos features se arman después, bootstrap conecta cada
// fuente una vez construidos todos (reglas §A.7, decisión D36).
package facts

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	riskuc "kaoru/internal/feature/risk/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Facts reúne las fuentes de hechos. Una fuente sin conectar responde «no hay»:
// sin RISK, sin TC y sin defectos, que es lo que bloquea por defecto.
type Facts struct {
	risk *riskuc.GetRisk
}

var _ port.Facts = (*Facts)(nil)

// New crea el adaptador sin fuentes.
func New() *Facts { return &Facts{} }

// ConnectRisk conecta el feature risk. Se llama una sola vez, al arrancar.
func (f *Facts) ConnectRisk(get *riskuc.GetRisk) { f.risk = get }

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

// Coverage todavía no tiene fuente: testdesign y execution llegan después.
func (f *Facts) Coverage(context.Context, shared.Code) (domain.CoverageFact, error) {
	return domain.CoverageFact{}, nil
}

// OpenDefects todavía no tiene fuente: defect llega después.
func (f *Facts) OpenDefects(context.Context, shared.Code) ([]domain.DefectFact, error) {
	return nil, nil
}
