// Package port declara lo que necesitan los casos de uso de component.
package port

import (
	"context"

	"kaoru/internal/feature/component/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota la lista de COMP. Un campo vacío no filtra; sin IncludeRetired, los retirados no salen.
type Filter struct {
	SystemKey      string
	Kind           string
	Query          string
	IncludeRetired bool
	Page           int
	Size           int
}

// ComponentView es un COMP con el nombre de quien lo creó, para mostrarlo.
type ComponentView struct {
	Component     domain.Component
	CreatedByName string
}

// Repository guarda y lee los COMP.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	// FindSystem devuelve NOT_FOUND_RESOURCE si el sistema no existe.
	FindSystem(ctx context.Context, key string) (domain.System, error)
	// Insert y Update devuelven domain.ErrNameTaken si el nombre ya existe en el sistema.
	Insert(ctx context.Context, component domain.Component) error
	Update(ctx context.Context, component domain.Component) error
	// Lock bloquea la entidad hasta el fin de la transacción en curso. Devuelve NOT_FOUND_RESOURCE si no existe.
	Lock(ctx context.Context, code shared.Code) error
	FindByCode(ctx context.Context, code shared.Code) (ComponentView, error)
	// FindByCodes devuelve los COMP que existen entre los códigos dados.
	FindByCodes(ctx context.Context, codes []string) ([]domain.Component, error)
	List(ctx context.Context, filter Filter) (components []ComponentView, total int, err error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un COMP.
type AuditRecord struct {
	Actor       shared.UserId
	ComponentId string
	Code        shared.Code
	Action      AuditAction
	Field       string
	OldValue    string
	NewValue    string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
