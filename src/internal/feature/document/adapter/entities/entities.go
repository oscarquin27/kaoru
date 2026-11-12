// Package entities implementa el puerto Entities con los casos de uso de los features
// cuyas entidades admiten documentos.
package entities

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	bundleuc "kaoru/internal/feature/bundle/application/usecase"
	changeuc "kaoru/internal/feature/change/application/usecase"
	defectuc "kaoru/internal/feature/defect/application/usecase"
	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/domain"
	environmentuc "kaoru/internal/feature/environment/application/usecase"
	executionuc "kaoru/internal/feature/execution/application/usecase"
	productionuc "kaoru/internal/feature/production/application/usecase"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	testdesignuc "kaoru/internal/feature/testdesign/application/usecase"
)

// Sources son los casos de uso con los que se busca cada tipo de entidad.
type Sources struct {
	Requirement *requirementuc.UseCases
	Change      *changeuc.UseCases
	TestDesign  *testdesignuc.UseCases
	Execution   *executionuc.UseCases
	Defect      *defectuc.UseCases
	Bundle      *bundleuc.UseCases
	Environment *environmentuc.UseCases
	Production  *productionuc.UseCases
}

// Entities resuelve la entidad a la que se enlaza un documento.
type Entities struct {
	s Sources
}

var _ port.Entities = (*Entities)(nil)

// New crea el adaptador.
func New(s Sources) *Entities { return &Entities{s: s} }

// Resolve conserva el código de error del feature dueño de la entidad.
func (e *Entities) Resolve(ctx context.Context, entityType, code string) (domain.EntityRef, error) {
	var id string
	var err error
	switch entityType {
	case "requirement":
		out, e2 := e.s.Requirement.GetRequirement.Execute(ctx, requirementuc.GetRequirementInput{Code: code})
		id, err = out.Id, e2
	case "change":
		out, e2 := e.s.Change.GetChange.Execute(ctx, changeuc.GetChangeInput{Code: code})
		id, err = out.Change.Id, e2
	case "functional_case":
		out, e2 := e.s.TestDesign.GetFunctionalCase.Execute(ctx, testdesignuc.GetFunctionalCaseInput{Code: code})
		id, err = out.Id, e2
	case "test_case":
		out, e2 := e.s.TestDesign.GetTestCase.Execute(ctx, testdesignuc.GetTestCaseInput{Code: code})
		id, err = out.Id, e2
	case "test_run":
		out, e2 := e.s.Execution.GetRun.Execute(ctx, executionuc.GetRunInput{Code: code})
		id, err = out.Id, e2
	case "exploratory_session":
		out, e2 := e.s.Execution.GetExploration.Execute(ctx, executionuc.GetExplorationInput{Code: code})
		id, err = out.Id, e2
	case "defect":
		out, e2 := e.s.Defect.GetDefect.Execute(ctx, defectuc.GetDefectInput{Code: code})
		id, err = out.Defect.Id, e2
	case "package":
		out, e2 := e.s.Bundle.FindPackage.Execute(ctx, bundleuc.FindPackageInput{Code: code})
		id, err = out.Id, e2
	case "deployment":
		out, e2 := e.s.Environment.GetDeployment.Execute(ctx, environmentuc.GetDeploymentInput{Code: code})
		id, err = out.Id, e2
	case "checklist":
		out, e2 := e.s.Production.GetChecklist.Execute(ctx, productionuc.GetChecklistInput{Code: code})
		id, err = out.Checklist.Id, e2
	case "diagnostic":
		out, e2 := e.s.Production.GetDiagnostic.Execute(ctx, productionuc.GetDiagnosticInput{Code: code})
		id, err = out.Id, e2
	case "smoke_test":
		out, e2 := e.s.Production.GetSmokeTest.Execute(ctx, productionuc.GetSmokeTestInput{Code: code})
		id, err = out.Id, e2
	default:
		return domain.EntityRef{}, axis.New(axis.NotFoundResource, "Ese tipo de entidad no admite documentos.")
	}
	if err != nil {
		return domain.EntityRef{}, err
	}
	return domain.EntityRef{Type: entityType, Id: id, Code: code}, nil
}
