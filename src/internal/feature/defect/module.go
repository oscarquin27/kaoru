// Package defect ensambla el feature de DEF.
package defect

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	"kaoru/internal/feature/defect/adapter/audit"
	"kaoru/internal/feature/defect/adapter/postgres"
	"kaoru/internal/feature/defect/adapter/testdesign"
	"kaoru/internal/feature/defect/adapter/workflow"
	"kaoru/internal/feature/defect/application/usecase"
	"kaoru/internal/feature/defect/delivery/api"
	testdesignuc "kaoru/internal/feature/testdesign/application/usecase"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database    *corepg.Database
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
	Workflow    *workflowuc.UseCases
	Audit       *audituc.UseCases
	TestDesign  *testdesignuc.UseCases
}

// Module expone los casos de uso y las rutas del feature.
type Module struct {
	UseCases *usecase.UseCases
	handler  *api.Handler
}

// New arma el módulo.
func New(d Deps) (*Module, error) {
	trail := audit.New(d.Audit)
	uc, err := usecase.New(usecase.Deps{
		Repository:  postgres.NewRepository(d.Database),
		Workflow:    workflow.New(d.Workflow),
		TestDesign:  testdesign.New(d.TestDesign.ListRequirementTestCases),
		Timeline:    trail,
		AuditTrail:  trail,
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
