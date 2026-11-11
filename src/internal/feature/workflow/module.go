// Package workflow ensambla el feature de flujos configurables.
package workflow

import (
	"github.com/gin-gonic/gin"

	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/workflow/adapter/postgres"
	"kaoru/internal/feature/workflow/application/usecase"
	"kaoru/internal/feature/workflow/delivery/api"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database *corepg.Database
}

// Module expone los casos de uso y las rutas del feature.
type Module struct {
	UseCases *usecase.UseCases
	handler  *api.Handler
}

// New arma el módulo.
func New(d Deps) (*Module, error) {
	uc, err := usecase.New(usecase.Deps{Repository: postgres.NewRepository(d.Database)})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc)}, nil
}

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
