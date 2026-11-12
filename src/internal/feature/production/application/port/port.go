// Package port declara lo que necesitan los casos de uso de production.
package port

import (
	"context"

	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota una lista por el código del PKG. Vacío no filtra.
type Filter struct {
	PackageCode string
	Page        int
	Size        int
}

// Repository guarda y lee plantillas, CHK, DIAG y SMOKE. Las búsquedas devuelven NOT_FOUND_RESOURCE
// si no hay fila. No tiene cómo editar ni borrar nada: todo es de solo inserción.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año para ese prefijo.
	NextCode(ctx context.Context, prefix string, year int) (shared.Code, error)
	FindSystem(ctx context.Context, key string) (domain.System, error)
	// LockPackage bloquea el PKG hasta el fin de la transacción en curso: es la fila que protege
	// las firmas de sus checklists y la revisión de sus DIAG.
	LockPackage(ctx context.Context, packageId string) error

	// NextTemplateVersion devuelve la versión que sigue a la última publicada del sistema.
	NextTemplateVersion(ctx context.Context, systemKey string) (int, error)
	// InsertTemplate devuelve DUPLICATE_RESOURCE si esa versión ya existe.
	InsertTemplate(ctx context.Context, template domain.Template) error
	LatestTemplate(ctx context.Context, systemKey string) (domain.Template, error)
	// ListTemplates devuelve todas las versiones, la más reciente de cada sistema primero.
	ListTemplates(ctx context.Context, systemKey string) ([]domain.Template, error)

	// FindPackage y FindEnvironment leen lo que un checklist referencia, con su sistema.
	FindPackage(ctx context.Context, code string) (domain.PackageRef, error)
	FindEnvironment(ctx context.Context, code string) (domain.EnvironmentRef, error)
	InsertChecklist(ctx context.Context, checklist domain.Checklist) error
	InsertSignature(ctx context.Context, checklistId string, signature domain.Signature) error
	// FindChecklist devuelve el CHK con sus pasos y sus firmas.
	FindChecklist(ctx context.Context, code shared.Code) (domain.Checklist, error)
	ListChecklists(ctx context.Context, filter Filter) (checklists []domain.Checklist, total int, err error)

	// FindDeployment lee el despliegue al que se refiere un DIAG o un SMOKE.
	FindDeployment(ctx context.Context, code string) (domain.DeploymentRef, error)
	InsertDiagnostic(ctx context.Context, diagnostic domain.Diagnostic) error
	InsertReview(ctx context.Context, diagnosticId string, review domain.Review) error
	FindDiagnostic(ctx context.Context, code shared.Code) (domain.Diagnostic, error)
	ListDiagnostics(ctx context.Context, filter Filter) (diagnostics []domain.Diagnostic, total int, err error)
	InsertSmokeTest(ctx context.Context, smokeTest domain.SmokeTest) error
	FindSmokeTest(ctx context.Context, code shared.Code) (domain.SmokeTest, error)
	ListSmokeTests(ctx context.Context, filter Filter) (smokeTests []domain.SmokeTest, total int, err error)

	// PackageFacts devuelve los hechos vacíos si el PKG no existe.
	PackageFacts(ctx context.Context, packageCode string) (domain.PackageFacts, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// Tipos de entidad que audita este feature.
const (
	EntityTemplate   = "checklist_template"
	EntityChecklist  = "checklist"
	EntityDiagnostic = "diagnostic"
	EntitySmokeTest  = "smoke_test"
)

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature. Una firma y una revisión son decisiones de quien las hace.
const (
	AuditCreate   AuditAction = "create"
	AuditDecision AuditAction = "decision"
)

// AuditRecord es un hecho sobre una plantilla, un CHK, un DIAG o un SMOKE.
// Una plantilla no tiene ID legible: Code va vacío.
type AuditRecord struct {
	Actor      shared.UserId
	EntityType string
	EntityId   string
	Code       string
	Action     AuditAction
	Field      string
	NewValue   string
	Reason     string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
