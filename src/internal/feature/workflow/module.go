// Package workflow ensambla el feature de flujos configurables.
package workflow

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	"kaoru/internal/feature/workflow/adapter/audit"
	"kaoru/internal/feature/workflow/adapter/postgres"
	"kaoru/internal/feature/workflow/application/usecase"
	"kaoru/internal/feature/workflow/delivery/api"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database *corepg.Database
	Clock    clock.Clock
}

// Module expone los casos de uso y las rutas del feature.
type Module struct {
	UseCases *usecase.UseCases
	handler  *api.Handler
	trail    *audit.Trail
}

// New arma el módulo.
func New(d Deps) (*Module, error) {
	trail := audit.New()
	uc, err := usecase.New(usecase.Deps{
		Repository: postgres.NewRepository(d.Database), AuditTrail: trail, UnitOfWork: d.Database, Clock: d.Clock,
	})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc), trail: trail}, nil
}

// ConnectAudit conecta el feature audit. bootstrap lo llama después de armar los dos módulos.
func (m *Module) ConnectAudit(uc *audituc.UseCases) { m.trail.Connect(uc.Record) }

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
