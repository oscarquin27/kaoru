// Package port declara lo que necesitan los casos de uso de build.
package port

import (
	"context"

	"kaoru/internal/feature/build/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota la lista de BUILD. Un campo vacío no filtra.
type Filter struct {
	SystemKey string
	// RequirementCode deja los BUILD que incluyen ese REQ en su manifiesto.
	RequirementCode string
	// Query es un texto contenido en la versión.
	Query string
	Page  int
	Size  int
}

// Repository guarda y lee los BUILD. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
// No tiene cómo editar ni borrar un BUILD.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	FindSystem(ctx context.Context, key string) (domain.System, error)
	// Insert devuelve DUPLICATE_RESOURCE si el checksum ya está registrado.
	Insert(ctx context.Context, build domain.Build) error
	FindByCode(ctx context.Context, code shared.Code) (domain.Build, error)
	// FindCodeByChecksum devuelve el código del BUILD que ya tiene ese checksum.
	FindCodeByChecksum(ctx context.Context, checksum string) (shared.Code, error)
	List(ctx context.Context, filter Filter) (builds []domain.Build, total int, err error)
}

// Requirements busca en el feature requirement los REQ del manifiesto.
type Requirements interface {
	// Find devuelve NOT_FOUND_RESOURCE si el REQ no existe.
	Find(ctx context.Context, code string) (domain.RequirementRef, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// AuditRecord es el hecho de haber registrado un BUILD.
type AuditRecord struct {
	Actor    shared.UserId
	BuildId  string
	Code     shared.Code
	Field    string
	NewValue string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
