// Package port declara lo que necesitan los casos de uso de comment.
package port

import (
	"context"

	"kaoru/internal/feature/comment/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee comentarios. Solo inserta.
type Repository interface {
	Insert(ctx context.Context, comment domain.Comment) error
	// List devuelve los comentarios de una entidad, del más antiguo al más reciente.
	List(ctx context.Context, entity domain.EntityRef) ([]domain.Comment, error)
	FindUserName(ctx context.Context, id shared.UserId) (string, error)
}

// Entities resuelve, en los demás features, la entidad que se comenta.
type Entities interface {
	// Resolve devuelve NOT_FOUND_RESOURCE si la entidad no existe o el tipo no admite comentarios.
	Resolve(ctx context.Context, entityType, code string) (domain.EntityRef, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	// RecordComment deja constancia de que se comentó una entidad.
	RecordComment(ctx context.Context, comment domain.Comment) error
}
