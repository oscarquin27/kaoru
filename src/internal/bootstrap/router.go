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
	"kaoru/internal/feature/build"
	"kaoru/internal/feature/bundle"
	"kaoru/internal/feature/change"
	"kaoru/internal/feature/comment"
	commententities "kaoru/internal/feature/comment/adapter/entities"
	"kaoru/internal/feature/component"
	"kaoru/internal/feature/defect"
	"kaoru/internal/feature/document"
	"kaoru/internal/feature/document/adapter/entities"
	"kaoru/internal/feature/environment"
	"kaoru/internal/feature/execution"
	"kaoru/internal/feature/regression"
	"kaoru/internal/feature/requirement"
	"kaoru/internal/feature/risk"
	"kaoru/internal/feature/savedview"
	"kaoru/internal/feature/testdesign"
	"kaoru/internal/feature/traceability"
	"kaoru/internal/feature/user"
	"kaoru/internal/feature/workflow"
)

// RouterDeps es lo que necesita el router: la infraestructura ya creada.
type RouterDeps struct {
	Config      Config
	Database    *corepg.Database
	FileStore   *filestore.Local
	Verifier    oidc.Verifier
	Metrics     *httpserver.Metrics
	Logger      logging.Logger
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// NewRouter construye los módulos en orden de dependencia y monta middlewares y rutas.
func NewRouter(d RouterDeps) (*gin.Engine, error) {
	workflowModule, err := workflow.New(workflow.Deps{Database: d.Database, Clock: d.Clock})
	if err != nil {
		return nil, err
	}
	auditModule, err := audit.New(audit.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator, Workflow: workflowModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	// audit depende de workflow para los permisos y workflow audita sus cambios de configuración:
	// el segundo sentido se conecta aquí, con los dos módulos ya armados (decisión D30).
	workflowModule.ConnectAudit(auditModule.UseCases)
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
	componentModule, err := component.New(component.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	requirementModule, err := requirement.New(requirement.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, Components: componentModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	buildModule, err := build.New(build.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, Requirements: requirementModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	environmentModule, err := environment.New(environment.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, Builds: buildModule.UseCases,
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
		Components: componentModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	executionModule, err := execution.New(execution.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, TestDesign: testdesignModule.UseCases,
		Builds: buildModule.UseCases, Environments: environmentModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	defectModule, err := defect.New(defect.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases, TestDesign: testdesignModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	regressionModule, err := regression.New(regression.Deps{
		Database: d.Database, Clock: d.Clock, Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
	})
	if err != nil {
		return nil, err
	}
	bundleModule, err := bundle.New(bundle.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
		Builds: buildModule.UseCases, Requirements: requirementModule.UseCases, Regression: regressionModule.UseCases,
		Defects: defectModule.UseCases, Environments: environmentModule.UseCases,
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
			Bundle: bundleModule.UseCases, Environment: environmentModule.UseCases,
		},
		AllowedExtensions: d.Config.Documents.AllowedExtensions,
		ConverterURL:      d.Config.Documents.ConverterURL,
	})
	if err != nil {
		return nil, err
	}
	commentModule, err := comment.New(comment.Deps{
		Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator,
		Workflow: workflowModule.UseCases, Audit: auditModule.UseCases,
		Entities: commententities.Sources{
			Requirement: requirementModule.UseCases, Change: changeModule.UseCases, Defect: defectModule.UseCases,
			Bundle: bundleModule.UseCases,
		},
	})
	if err != nil {
		return nil, err
	}
	traceabilityModule, err := traceability.New(traceability.Deps{
		Database: d.Database, Clock: d.Clock, Requirements: requirementModule.UseCases, Execution: executionModule.UseCases,
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
		d.Metrics.Middleware(),
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
	componentModule.RegisterAPI(api)
	savedviewModule, err := savedview.New(savedview.Deps{Database: d.Database, Clock: d.Clock, IdGenerator: d.IdGenerator})
	if err != nil {
		return nil, err
	}
	requirementModule.RegisterAPI(api)
	riskModule.RegisterAPI(api)
	testdesignModule.RegisterAPI(api)
	buildModule.RegisterAPI(api)
	environmentModule.RegisterAPI(api)
	executionModule.RegisterAPI(api)
	defectModule.RegisterAPI(api)
	bundleModule.RegisterAPI(api)
	regressionModule.RegisterAPI(api)
	documentModule.RegisterAPI(api)
	commentModule.RegisterAPI(api)
	savedviewModule.RegisterAPI(api)
	traceabilityModule.RegisterAPI(api)
	return engine, nil
}
