// Package testdesign ensambla el feature de diseño de pruebas (UC y TC).
package testdesign

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	componentuc "kaoru/internal/feature/component/application/usecase"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/testdesign/adapter/audit"
	"kaoru/internal/feature/testdesign/adapter/component"
	"kaoru/internal/feature/testdesign/adapter/postgres"
	"kaoru/internal/feature/testdesign/adapter/requirement"
	"kaoru/internal/feature/testdesign/adapter/workflow"
	"kaoru/internal/feature/testdesign/application/usecase"
	"kaoru/internal/feature/testdesign/delivery/api"
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
	Components   *componentuc.UseCases
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
		Components:   component.New(d.Components.ResolveComponents),
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
