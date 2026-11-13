// Package bundle ensambla el feature de PKG. Se llama bundle porque package es palabra reservada de Go.
package bundle

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	builduc "kaoru/internal/feature/build/application/usecase"
	"kaoru/internal/feature/bundle/adapter/audit"
	"kaoru/internal/feature/bundle/adapter/build"
	"kaoru/internal/feature/bundle/adapter/facts"
	"kaoru/internal/feature/bundle/adapter/postgres"
	"kaoru/internal/feature/bundle/adapter/regression"
	"kaoru/internal/feature/bundle/adapter/requirement"
	"kaoru/internal/feature/bundle/adapter/workflow"
	"kaoru/internal/feature/bundle/application/usecase"
	"kaoru/internal/feature/bundle/delivery/api"
	defectuc "kaoru/internal/feature/defect/application/usecase"
	environmentuc "kaoru/internal/feature/environment/application/usecase"
	productionuc "kaoru/internal/feature/production/application/usecase"
	regressionuc "kaoru/internal/feature/regression/application/usecase"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database     *corepg.Database
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
	Workflow     *workflowuc.UseCases
	Audit        *audituc.UseCases
	Builds       *builduc.UseCases
	Requirements *requirementuc.UseCases
	Regression   *regressionuc.UseCases
	Defects      *defectuc.UseCases
	Environments *environmentuc.UseCases
	Production   *productionuc.UseCases
}

// Module expone los casos de uso y las rutas del feature.
type Module struct {
	UseCases *usecase.UseCases
	handler  *api.Handler
}

// New arma el módulo.
func New(d Deps) (*Module, error) {
	uc, err := usecase.New(usecase.Deps{
		Repository:   postgres.NewRepository(d.Database),
		Workflow:     workflow.New(d.Workflow),
		Builds:       build.New(d.Builds.FindBuild),
		Requirements: requirement.New(d.Requirements.GetRequirement, d.Requirements.TransitionRequirement),
		Regression:   regression.New(d.Regression.ReopenRegression),
		Facts: facts.New(facts.Sources{
			Regression: d.Regression.GetRegressionFacts, Defects: d.Defects.ListOpenDefects,
			Deployments: d.Environments.HasProductionDeployment,
			Production:  d.Production.GetPackageFacts, Requirement: d.Requirements.GetRequirement,
		}),
		AuditTrail:  audit.New(d.Audit.Record),
		UnitOfWork:  d.Database,
		Clock:       d.Clock,
		IdGenerator: d.IdGenerator,
	})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc)}, nil
}

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
