// Package traceability ensambla el feature de solo lectura que cruza todos los demás:
// la ficha consolidada del REQ, la matriz, la búsqueda global y el catálogo de sistemas.
package traceability

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	corepg "kaoru/internal/core/pkg/postgres"
	executionuc "kaoru/internal/feature/execution/application/usecase"
	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/traceability/adapter/execution"
	"kaoru/internal/feature/traceability/adapter/postgres"
	"kaoru/internal/feature/traceability/adapter/requirement"
	"kaoru/internal/feature/traceability/application/usecase"
	"kaoru/internal/feature/traceability/delivery/api"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database     *corepg.Database
	Clock        clock.Clock
	Requirements *requirementuc.UseCases
	Execution    *executionuc.UseCases
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
		Readiness:  requirement.New(d.Requirements.GetReadiness),
		Coverage:   execution.New(d.Execution.GetCoverage),
		Clock:      d.Clock,
	})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc)}, nil
}

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
