// Package port declara lo que necesitan los casos de uso de environment.
package port

import (
	"context"

	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// EnvironmentFilter acota la lista de ENV. Un campo vacío no filtra.
type EnvironmentFilter struct {
	SystemKey      string
	Kind           string
	IncludeRetired bool
	Page           int
	Size           int
}

// DeploymentFilter acota la lista de DEP por código de ENV, de BUILD o de PKG. Un campo vacío no filtra.
type DeploymentFilter struct {
	EnvironmentCode string
	BuildCode       string
	PackageCode     string
	Page            int
	Size            int
}

// EnvironmentView es un ENV con lo que tiene instalado; Installed es nil si nunca hubo un despliegue exitoso.
type EnvironmentView struct {
	Environment domain.Environment
	Installed   *domain.Installed
}

// Repository guarda y lee ENV y DEP. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
// No tiene cómo editar ni borrar un DEP.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año para ese prefijo.
	NextCode(ctx context.Context, prefix string, year int) (shared.Code, error)
	FindSystem(ctx context.Context, key string) (domain.System, error)

	// InsertEnvironment y UpdateEnvironment devuelven DUPLICATE_RESOURCE si el nombre ya existe en el sistema.
	InsertEnvironment(ctx context.Context, environment domain.Environment) error
	UpdateEnvironment(ctx context.Context, environment domain.Environment) error
	// LockEnvironment bloquea el ENV hasta el fin de la transacción en curso.
	LockEnvironment(ctx context.Context, code shared.Code) error
	FindEnvironment(ctx context.Context, code shared.Code) (EnvironmentView, error)
	ListEnvironments(ctx context.Context, filter EnvironmentFilter) (environments []EnvironmentView, total int, err error)

	// FindPackage lee un PKG por su código: es el sentido de la llave foránea del despliegue.
	FindPackage(ctx context.Context, code string) (domain.PackageRef, error)
	InsertDeployment(ctx context.Context, deployment domain.Deployment) error
	FindDeployment(ctx context.Context, code shared.Code) (domain.Deployment, error)
	ListDeployments(ctx context.Context, filter DeploymentFilter) (deployments []domain.Deployment, total int, err error)
	// HasProductionDeployment dice si ese PKG tiene un despliegue exitoso en un ambiente de producción.
	HasProductionDeployment(ctx context.Context, packageCode string) (bool, error)
}

// Builds busca en el feature build el BUILD que se despliega.
type Builds interface {
	// Find devuelve NOT_FOUND_RESOURCE si el BUILD no existe.
	Find(ctx context.Context, code string) (domain.BuildRef, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// Tipos de entidad que audita este feature.
const (
	EntityEnvironment = "environment"
	EntityDeployment  = "deployment"
)

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un ENV o un DEP.
type AuditRecord struct {
	Actor      shared.UserId
	EntityType string
	EntityId   string
	Code       shared.Code
	Action     AuditAction
	Field      string
	OldValue   string
	NewValue   string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
