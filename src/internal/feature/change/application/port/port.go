// Package port declara lo que necesitan los casos de uso de change.
package port

import (
	"context"

	"kaoru/internal/feature/change/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota la lista de CHG. Un campo vacío no filtra.
type Filter struct {
	SystemKey string
	StateKey  string
	Query     string
	Page      int
	Size      int
}

// ChangeView es un CHG con el nombre de quien lo creó, para mostrarlo.
type ChangeView struct {
	Change        domain.Change
	CreatedByName string
}

// Repository guarda y lee los CHG.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	// FindSystem devuelve NOT_FOUND_RESOURCE si el sistema no existe.
	FindSystem(ctx context.Context, key string) (domain.System, error)
	Insert(ctx context.Context, change domain.Change) error
	Update(ctx context.Context, change domain.Change) error
	// Lock bloquea la entidad hasta el fin de la transacción en curso. Devuelve NOT_FOUND_RESOURCE si no existe.
	Lock(ctx context.Context, code shared.Code) error
	FindByCode(ctx context.Context, code shared.Code) (ChangeView, error)
	List(ctx context.Context, filter Filter) (changes []ChangeView, total int, err error)
}

// Transition es una transición que el flujo permite a un actor.
type Transition struct {
	To             domain.State
	RequiresReason bool
}

// Workflow responde por los permisos y los estados del flujo del CHG.
type Workflow interface {
	Require(ctx context.Context, roles []string, permission string) error
	InitialState(ctx context.Context) (domain.State, error)
	ResolveTransition(ctx context.Context, from, to string, roles []string) (Transition, error)
}

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate     AuditAction = "create"
	AuditUpdate     AuditAction = "update"
	AuditTransition AuditAction = "transition"
)

// AuditRecord es un hecho sobre un CHG.
type AuditRecord struct {
	Actor    shared.UserId
	ChangeId string
	Code     shared.Code
	Action   AuditAction
	Field    string
	OldValue string
	NewValue string
	Reason   string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
