// Package metrics ensambla el feature de solo lectura de las métricas: cobertura por REQ, tiempo en
// estado de los DEF, tamaño de la regresión y defectos escapados a producción.
package metrics

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/metrics/adapter/postgres"
	"kaoru/internal/feature/metrics/application/usecase"
	"kaoru/internal/feature/metrics/delivery/api"
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
}

// New arma el módulo.
func New(d Deps) (*Module, error) {
	uc, err := usecase.New(usecase.Deps{Repository: postgres.NewRepository(d.Database), Clock: d.Clock})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc)}, nil
}

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
