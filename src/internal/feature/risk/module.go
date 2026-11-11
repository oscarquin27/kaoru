// Package risk ensambla el feature de RISK.
package risk

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/risk/adapter/audit"
	"kaoru/internal/feature/risk/adapter/postgres"
	"kaoru/internal/feature/risk/adapter/requirement"
	"kaoru/internal/feature/risk/adapter/workflow"
	"kaoru/internal/feature/risk/application/usecase"
	"kaoru/internal/feature/risk/delivery/api"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database     *corepg.Database
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
	Workflow     *workflowuc.UseCases
	Audit        *audituc.UseCases
	Requirements *requirementuc.UseCases
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
		Requirements: requirement.New(d.Requirements.GetRequirement),
		Permissions:  workflow.NewPermissions(d.Workflow.CheckPermission),
		AuditTrail:   audit.New(d.Audit.Record),
		UnitOfWork:   d.Database,
		Clock:        d.Clock,
		IdGenerator:  d.IdGenerator,
	})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc)}, nil
}

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
