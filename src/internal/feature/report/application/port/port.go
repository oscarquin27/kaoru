// Package port declara lo que necesitan los casos de uso de report.
package port

import (
	"context"

	"kaoru/internal/feature/report/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota la lista de RPT por el código del PKG. Vacío no filtra.
type Filter struct {
	PackageCode string
	Page        int
	Size        int
}

// Repository guarda y lee los RPT. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
// No tiene cómo editar ni borrar un RPT: es de solo inserción (regla de negocio 11).
type Repository interface {
	// NextCode reserva el siguiente ID legible del año para ese prefijo.
	NextCode(ctx context.Context, prefix string, year int) (shared.Code, error)
	FindPackage(ctx context.Context, code string) (domain.PackageRef, error)
	// LockPackage bloquea el PKG hasta el fin de la transacción: dos emisiones a la vez no pueden
	// tomar la misma versión.
	LockPackage(ctx context.Context, packageId string) error
	// Latest devuelve la versión vigente del RPT del PKG.
	Latest(ctx context.Context, packageId string) (domain.Report, error)
	Insert(ctx context.Context, report domain.Report) error
	// FindByCode devuelve el RPT con su contenido.
	FindByCode(ctx context.Context, code shared.Code) (domain.Report, error)
	// List devuelve los RPT sin su contenido, del más reciente al más antiguo.
	List(ctx context.Context, filter Filter) (reports []domain.Report, total int, err error)
	UserName(ctx context.Context, id shared.UserId) (string, error)
}

// Evidence reúne lo que el resto de Kaoru sabe del PKG. Devuelve el contenido sin los datos de la emisión.
type Evidence interface {
	Gather(ctx context.Context, packageCode string) (domain.Content, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// EntityReport es el tipo de entidad que audita este feature.
const EntityReport = "conformance_report"

// AuditRecord es la emisión de un RPT.
type AuditRecord struct {
	Actor    shared.UserId
	EntityId string
	Code     string
	Field    string
	NewValue string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
