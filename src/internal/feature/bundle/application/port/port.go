// Package port declara lo que necesitan los casos de uso de bundle.
package port

import (
	"context"
	"time"

	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota la lista de PKG. Un campo vacío no filtra.
type Filter struct {
	SystemKey string
	StateKey  string
	Query     string
	Page      int
	Size      int
}

// Repository guarda y lee los PKG con su contenido.
// Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	FindSystem(ctx context.Context, key string) (domain.System, error)

	Insert(ctx context.Context, pkg domain.Package) error
	// Update guarda el título, la descripción, el estado y la fecha de cambio; el contenido va aparte.
	Update(ctx context.Context, pkg domain.Package) error
	// Lock bloquea la entidad hasta el fin de la transacción en curso. Devuelve NOT_FOUND_RESOURCE si no existe.
	Lock(ctx context.Context, code shared.Code) error
	// FindByCode devuelve el PKG con sus BUILD y sus REQ.
	FindByCode(ctx context.Context, code shared.Code) (domain.Package, error)
	List(ctx context.Context, filter Filter) (packages []domain.Package, total int, err error)

	AddBuild(ctx context.Context, packageId, buildId string, by shared.UserId, at time.Time) error
	RemoveBuild(ctx context.Context, packageId, buildId string) error
	AddRequirement(ctx context.Context, packageId, requirementId string, by shared.UserId, at time.Time) error
	RemoveRequirement(ctx context.Context, packageId, requirementId string) error
}

// Builds busca en el feature build el BUILD que entra al PKG.
type Builds interface {
	// Find devuelve NOT_FOUND_RESOURCE si el BUILD no existe.
	Find(ctx context.Context, code string) (domain.Build, error)
}

// Requirements busca en el feature requirement el REQ que entra al PKG y lo pasa a «En paquete».
type Requirements interface {
	// Find devuelve NOT_FOUND_RESOURCE si el REQ no existe.
	Find(ctx context.Context, code string) (domain.Requirement, error)
	// MoveToPackage pasa el REQ a «En paquete» con el mismo actor, dentro de la transacción en curso.
	// Devuelve PERMISSION_DENIED si el actor no tiene el permiso de esa transición.
	MoveToPackage(ctx context.Context, actor shared.Actor, code string) error
}

// Regression avisa al feature regression de que la selección confirmada del PKG ya no corresponde.
type Regression interface {
	// Reopen quita la confirmación, si la hay, dentro de la transacción en curso.
	Reopen(ctx context.Context, actor shared.UserId, packageCode shared.Code, reason string) error
}

// Facts trae de otros features los hechos que las reglas de bloqueo necesitan.
type Facts interface {
	Regression(ctx context.Context, packageCode shared.Code) (domain.RegressionFact, error)
	// OpenDefects devuelve los defectos abiertos de esos REQ.
	OpenDefects(ctx context.Context, requirementCodes []string) ([]domain.DefectFact, error)
	// DeployedToProduction dice si el PKG tiene un despliegue exitoso en un ambiente de producción.
	DeployedToProduction(ctx context.Context, packageCode shared.Code) (bool, error)
}

// Transition es una transición del flujo del PKG con sus reglas activas.
type Transition struct {
	To             domain.State
	RequiresReason bool
	Rules          []domain.Rule
}

// Workflow responde por los permisos y los estados del flujo del PKG.
type Workflow interface {
	Require(ctx context.Context, roles []string, permission string) error
	InitialState(ctx context.Context) (domain.State, error)
	// ResolveTransition comprueba que el flujo permita el paso y que el rol tenga su permiso.
	ResolveTransition(ctx context.Context, from, to string, roles []string) (Transition, error)
	// Outgoing devuelve las transiciones que salen de un estado, sin mirar permisos.
	Outgoing(ctx context.Context, from string) ([]Transition, error)
}

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate     AuditAction = "create"
	AuditUpdate     AuditAction = "update"
	AuditTransition AuditAction = "transition"
)

// AuditRecord es un hecho sobre un PKG.
type AuditRecord struct {
	Actor     shared.UserId
	PackageId string
	Code      shared.Code
	Action    AuditAction
	Field     string
	OldValue  string
	NewValue  string
	Reason    string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
