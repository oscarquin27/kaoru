// Package report ensambla el feature del informe de conformidad (RPT-) de un PKG.
package report

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	bundleuc "kaoru/internal/feature/bundle/application/usecase"
	productionuc "kaoru/internal/feature/production/application/usecase"
	regressionuc "kaoru/internal/feature/regression/application/usecase"
	"kaoru/internal/feature/report/adapter/audit"
	"kaoru/internal/feature/report/adapter/evidence"
	"kaoru/internal/feature/report/adapter/postgres"
	"kaoru/internal/feature/report/adapter/workflow"
	"kaoru/internal/feature/report/application/usecase"
	"kaoru/internal/feature/report/delivery/api"
	traceabilityuc "kaoru/internal/feature/traceability/application/usecase"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database     *corepg.Database
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
	Workflow     *workflowuc.UseCases
	Audit        *audituc.UseCases
	Bundle       *bundleuc.UseCases
	Traceability *traceabilityuc.UseCases
	Regression   *regressionuc.UseCases
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
		Repository: postgres.NewRepository(d.Database),
		Evidence: evidence.New(evidence.Sources{
			Package: d.Bundle.GetPackage, Trace: d.Traceability.GetPackageTrace, Regression: d.Regression.GetRegression,
			Checklists: d.Production.ListChecklists, Diagnostics: d.Production.ListDiagnostics, SmokeTests: d.Production.ListSmokeTests,
		}),
		Permissions: workflow.NewPermissions(d.Workflow.CheckPermission),
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
