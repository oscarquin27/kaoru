// Package port declara lo que necesitan los casos de uso de system.
package port

import (
	"context"

	"kaoru/internal/feature/system/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee los sistemas. La lista la sirve traceability; aquí solo se escribe.
// Un sistema no se borra: los CHG, REQ, TC y demás lo referencian.
type Repository interface {
	// Insert devuelve DUPLICATE_RESOURCE si la clave ya existe.
	Insert(ctx context.Context, system domain.System) error
	Update(ctx context.Context, system domain.System) error
	// Lock bloquea el sistema hasta el fin de la transacción en curso y lo lee.
	// Devuelve NOT_FOUND_RESOURCE si no existe.
	Lock(ctx context.Context, key string) (domain.System, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// EntitySystem es el tipo de entidad que audita este feature.
const EntitySystem = "system"

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un sistema. El código que se audita es su clave.
type AuditRecord struct {
	Actor    shared.UserId
	EntityId string
	Key      string
	Action   AuditAction
	Field    string
	OldValue string
	NewValue string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
