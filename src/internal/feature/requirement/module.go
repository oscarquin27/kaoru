// Package requirement ensambla el feature de REQ.
package requirement

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	defectuc "kaoru/internal/feature/defect/application/usecase"
	executionuc "kaoru/internal/feature/execution/application/usecase"
	"kaoru/internal/feature/requirement/adapter/audit"
	"kaoru/internal/feature/requirement/adapter/facts"
	"kaoru/internal/feature/requirement/adapter/postgres"
	"kaoru/internal/feature/requirement/adapter/workflow"
	"kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/requirement/delivery/api"
	riskuc "kaoru/internal/feature/risk/application/usecase"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database    *corepg.Database
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
	Workflow    *workflowuc.UseCases
	Audit       *audituc.UseCases
}

// Module expone los casos de uso y las rutas del feature.
type Module struct {
	UseCases *usecase.UseCases
	handler  *api.Handler
	facts    *facts.Facts
}

// New arma el módulo. Las fuentes de hechos de otros features se conectan después.
func New(d Deps) (*Module, error) {
	sources := facts.New()
	uc, err := usecase.New(usecase.Deps{
		Repository:  postgres.NewRepository(d.Database),
		Workflow:    workflow.New(d.Workflow),
		Facts:       sources,
		AuditTrail:  audit.New(d.Audit.Record),
		UnitOfWork:  d.Database,
		Clock:       d.Clock,
		IdGenerator: d.IdGenerator,
	})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc), facts: sources}, nil
}

// ConnectRisk conecta el feature risk como fuente de hechos de las reglas de bloqueo.
// bootstrap lo llama después de armar los dos módulos, porque risk depende de requirement.
func (m *Module) ConnectRisk(uc *riskuc.UseCases) { m.facts.ConnectRisk(uc.GetRisk) }

// ConnectExecution conecta el feature execution como fuente de la cobertura.
func (m *Module) ConnectExecution(uc *executionuc.UseCases) { m.facts.ConnectCoverage(uc.GetCoverage) }

// ConnectDefect conecta el feature defect como fuente de los defectos abiertos.
func (m *Module) ConnectDefect(uc *defectuc.UseCases) { m.facts.ConnectDefects(uc.ListOpenDefects) }

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
