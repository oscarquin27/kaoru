package bootstrap

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/filestore"
	"kaoru/internal/core/pkg/httpserver"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/logging"
	"kaoru/internal/core/pkg/oidc"
	corepg "kaoru/internal/core/pkg/postgres"
	"kaoru/internal/core/pkg/problem"
	"kaoru/internal/feature/audit"
	"kaoru/internal/feature/change"
	"kaoru/internal/feature/defect"
	"kaoru/internal/feature/document"
	"kaoru/internal/feature/document/adapter/entities"
	"kaoru/internal/feature/execution"
	"kaoru/internal/feature/requirement"
	"kaoru/internal/feature/risk"
	"kaoru/internal/feature/testdesign"
	"kaoru/internal/feature/user"
	"kaoru/internal/feature/workflow"
)

// RouterDeps es lo que necesita el router: la infraestructura ya creada.
type RouterDeps struct {
	Config      Config
	Database    *corepg.Database
	FileStore   *filestore.Local
	Verifier    oidc.Verifier
	Logger      logging.Logger
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// NewRouter construye los módulos en orden de dependencia y monta middlewares y rutas.
func NewRouter(d RouterDeps) (*gin.Engine, error) {
	workflowModule, err := workflow.New(workflow.Deps{Database: d.Database})
	if err != nil {
		return nil, err
	}
	auditModule, err := audit.New(audit.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator, Workflow: workflowModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	userModule, err := user.New(user.Deps{
		Database: d.Database, Clock: d.Clock, Workflow: workflowModule.UseCases,
		LogoutURL: d.Config.OIDC.LogoutURL, AccountURL: d.Config.OIDC.AccountURL,
	})
	if err != nil {
		return nil, err
	}
	changeModule, err := change.New(change.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	requirementModule, err := requirement.New(requirement.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	riskModule, err := risk.New(risk.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, Requirements: requirementModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	testdesignModule, err := testdesign.New(testdesign.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, Requirements: requirementModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	executionModule, err := execution.New(execution.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, TestDesign: testdesignModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	defectModule, err := defect.New(defect.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	documentModule, err := document.New(document.Deps{
		Database: d.Database, FileStore: d.FileStore, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
		Entities: entities.Sources{
			Requirement: requirementModule.UseCases, Change: changeModule.UseCases, TestDesign: testdesignModule.UseCases,
			Execution: executionModule.UseCases, Defect: defectModule.UseCases,
		},
		AllowedExtensions: d.Config.Documents.AllowedExtensions,
	})
	if err != nil {
		return nil, err
	}
	// Las reglas de bloqueo del REQ necesitan hechos de features que dependen de él:
	// se conectan después de armar todos (reglas §A.7, decisión D36).
	requirementModule.ConnectRisk(riskModule.UseCases)
	requirementModule.ConnectExecution(executionModule.UseCases)
	requirementModule.ConnectDefect(defectModule.UseCases)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	if err := engine.SetTrustedProxies(d.Config.HTTP.TrustedProxies); err != nil {
		return nil, fmt.Errorf("http.trustedProxies: %w", err)
	}
	engine.Use(
		httpserver.RequestID(),
		httpserver.AccessLog(d.Logger),
		httpserver.Recovery(d.Logger),
		httpserver.MaxBody(d.Config.HTTP.MaxBodyBytes, d.Config.Documents.MaxBytes),
		httpserver.Origin(d.Config.HTTP.PublicOrigin),
	)
	engine.NoRoute(func(c *gin.Context) {
		problem.Write(c, axis.New(axis.NotFoundResource, "La ruta no existe."))
	})

	api := engine.Group("", oidc.Middleware(d.Verifier), userModule.EnsureProfile())
	userModule.RegisterAPI(api)
	workflowModule.RegisterAPI(api)
	auditModule.RegisterAPI(api)
	changeModule.RegisterAPI(api)
	requirementModule.RegisterAPI(api)
	riskModule.RegisterAPI(api)
	testdesignModule.RegisterAPI(api)
	executionModule.RegisterAPI(api)
	defectModule.RegisterAPI(api)
	documentModule.RegisterAPI(api)
	return engine, nil
}
