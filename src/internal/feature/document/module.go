// Package document ensambla el feature de documentos adjuntos.
package document

import (
	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/clock"
	corefs "kaoru/internal/core/pkg/filestore"
	"kaoru/internal/core/pkg/idgen"
	corepg "kaoru/internal/core/pkg/postgres"
	audituc "kaoru/internal/feature/audit/application/usecase"
	"kaoru/internal/feature/document/adapter/audit"
	"kaoru/internal/feature/document/adapter/entities"
	"kaoru/internal/feature/document/adapter/filestore"
	"kaoru/internal/feature/document/adapter/postgres"
	"kaoru/internal/feature/document/adapter/workflow"
	"kaoru/internal/feature/document/application/usecase"
	"kaoru/internal/feature/document/delivery/api"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Deps son las dependencias del módulo.
type Deps struct {
	Database          *corepg.Database
	FileStore         *corefs.Local
	Clock             clock.Clock
	IdGenerator       idgen.IdGenerator
	Workflow          *workflowuc.UseCases
	Audit             *audituc.UseCases
	Entities          entities.Sources
	AllowedExtensions []string
}

// Module expone los casos de uso y las rutas del feature.
type Module struct {
	UseCases *usecase.UseCases
	handler  *api.Handler
}

// New arma el módulo.
func New(d Deps) (*Module, error) {
	uc, err := usecase.New(usecase.Deps{
		Repository:        postgres.NewRepository(d.Database),
		FileStore:         filestore.New(d.FileStore),
		Entities:          entities.New(d.Entities),
		Permissions:       workflow.NewPermissions(d.Workflow.CheckPermission),
		AuditTrail:        audit.New(d.Audit.Record),
		UnitOfWork:        d.Database,
		Clock:             d.Clock,
		IdGenerator:       d.IdGenerator,
		AllowedExtensions: d.AllowedExtensions,
	})
	if err != nil {
		return nil, err
	}
	return &Module{UseCases: uc, handler: api.NewHandler(uc)}, nil
}

// RegisterAPI monta las rutas del feature.
func (m *Module) RegisterAPI(r gin.IRouter) { m.handler.Register(r) }
