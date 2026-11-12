// Package port declara lo que necesitan los casos de uso de planning.
package port

import (
	"context"

	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// Estados por los que se filtra la lista de hitos. Vacío no filtra.
const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
)

// Filter acota la lista de hitos.
type Filter struct {
	Status string
	Page   int
	Size   int
}

// Repository guarda y lee los hitos. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
// Un hito no se borra.
type Repository interface {
	Insert(ctx context.Context, milestone domain.Milestone) error
	Update(ctx context.Context, milestone domain.Milestone) error
	// Lock bloquea el hito hasta el fin de la transacción en curso. Un id mal formado tampoco existe.
	Lock(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (domain.Milestone, error)
	// List ordena por fecha, la más próxima primero.
	List(ctx context.Context, filter Filter) (milestones []domain.Milestone, total int, err error)
	// Pending devuelve todos los hitos sin cumplir, para las alertas.
	Pending(ctx context.Context) ([]domain.Milestone, error)
	// FindUser lee el responsable de un hito.
	FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error)
}

// Changes lee las ventanas de los CHG vigentes.
type Changes interface {
	Windows(ctx context.Context) ([]domain.Window, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// EntityMilestone es el tipo de entidad que audita este feature.
const EntityMilestone = "milestone"

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un hito. Un hito no tiene ID legible.
type AuditRecord struct {
	Actor    shared.UserId
	EntityId string
	Action   AuditAction
	Field    string
	OldValue string
	NewValue string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
