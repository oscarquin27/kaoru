// Package user ensambla el feature de perfil de usuario.
package user

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/feature/user/adapter/postgres"
	"kaoru/internal/feature/user/adapter/workflow"
	"kaoru/internal/feature/user/application/usecase"
	"kaoru/internal/feature/user/delivery/api"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database   *corepg.Database
	Clock      clock.Clock
	Workflow   *workflowuc.UseCases
	LogoutURL  string
	AccountURL string
}

// Module expone los casos de uso y las rutas del feature.
type Module struct {
	UseCases *usecase.UseCases
	handler  *api.Handler
}

// New arma el módulo.
func New(d Deps) (*Module, error) {
	uc, err := usecase.New(usecase.Deps{
		Repository:  postgres.NewRepository(d.Database),
		Permissions: workflow.NewPermissions(d.Workflow.ListPermissions),
		Clock:       d.Clock,
		LogoutURL:   d.LogoutURL,
		AccountURL:  d.AccountURL,
	})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc)}, nil
}

// EnsureProfile es el middleware que guarda el perfil del token.
func (m *Module) EnsureProfile() gin.HandlerFunc { return m.handler.EnsureProfile() }

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
