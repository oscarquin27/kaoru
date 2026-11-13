// Package facts implementa el puerto Facts con los casos de uso de los features de los que
// dependen las reglas de bloqueo del PKG: regression, defect, environment, production y requirement.
package facts

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	defectuc "kaoru/internal/feature/defect/application/usecase"
	environmentuc "kaoru/internal/feature/environment/application/usecase"
	productionuc "kaoru/internal/feature/production/application/usecase"
	regressionuc "kaoru/internal/feature/regression/application/usecase"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	shared "kaoru/internal/shared/domain"
)

// Sources son los casos de uso que dan cada hecho.
type Sources struct {
	Regression  *regressionuc.GetRegressionFacts
	Defects     *defectuc.ListOpenDefects
	Deployments *environmentuc.HasProductionDeployment
	Production  *productionuc.GetPackageFacts
	Requirement *requirementuc.GetRequirement
}

// Facts reúne los hechos de las reglas del PKG.
type Facts struct {
	s Sources
}

var _ port.Facts = (*Facts)(nil)

// New crea el adaptador.
func New(s Sources) *Facts { return &Facts{s: s} }

// Regression dice si la regresión del PKG existe, si está confirmada y qué TC le faltan.
func (f *Facts) Regression(ctx context.Context, packageCode shared.Code) (domain.RegressionFact, error) {
	out, err := f.s.Regression.Execute(ctx, regressionuc.GetRegressionFactsInput{PackageCode: packageCode.String()})
	if err != nil {
		return domain.RegressionFact{}, err
	}
	return domain.RegressionFact{Exists: out.Exists, Confirmed: out.Confirmed, Unpassed: out.Unpassed}, nil
}

// OpenDefects devuelve los defectos de esos REQ que no están cerrados.
func (f *Facts) OpenDefects(ctx context.Context, requirementCodes []string) ([]domain.DefectFact, error) {
	var out []domain.DefectFact
	for _, code := range requirementCodes {
		open, err := f.s.Defects.Execute(ctx, defectuc.ListOpenDefectsInput{RequirementCode: code})
		if err != nil {
			return nil, err
		}
		for _, d := range open {
			out = append(out, domain.DefectFact{Code: d.Code, Severity: d.Severity})
		}
	}
	return out, nil
}

// DeployedToProduction dice si el PKG tiene un despliegue exitoso en un ambiente de producción.
func (f *Facts) DeployedToProduction(ctx context.Context, packageCode shared.Code) (bool, error) {
	return f.s.Deployments.Execute(ctx, environmentuc.HasProductionDeploymentInput{PackageCode: packageCode.String()})
}

// Production dice cómo están el último checklist, el último DIAG y el último SMOKE del PKG.
func (f *Facts) Production(ctx context.Context, packageCode shared.Code) (domain.ProductionFact, error) {
	out, err := f.s.Production.Execute(ctx, productionuc.GetPackageFactsInput{PackageCode: packageCode.String()})
	if err != nil {
		return domain.ProductionFact{}, err
	}
	return domain.ProductionFact{
		ChecklistExists: out.ChecklistExists, ChecklistPending: out.ChecklistPending,
		Diagnostic: out.Diagnostic, Smoke: out.Smoke,
	}, nil
}

// Routes devuelve la ruta de cada REQ: la de su CHG o, si no tiene, la de su sistema.
func (f *Facts) Routes(ctx context.Context, requirementCodes []string) ([]string, error) {
	routes := make([]string, len(requirementCodes))
	for i, code := range requirementCodes {
		s, err := f.s.Requirement.Execute(ctx, requirementuc.GetRequirementInput{Code: code})
		if err != nil {
			return nil, err
		}
		routes[i] = s.System.DefaultRoute
		if s.Change != nil {
			routes[i] = s.Change.Route
		}
	}
	return routes, nil
}
