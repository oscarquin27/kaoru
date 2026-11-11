// Package port declara lo que necesitan los casos de uso de defect.
package port

import (
	"context"
	"time"

	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota la lista de DEF. Un campo vacío no filtra.
type Filter struct {
	StateKey        string
	RequirementCode string
	Severity        string
	AssigneeId      string
	Query           string
	// OnlyOpen deja fuera los DEF en un estado final.
	OnlyOpen bool
	Page     int
	Size     int
}

// Repository guarda y lee los DEF. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	FindRequirement(ctx context.Context, code string) (domain.RequirementRef, error)
	FindRun(ctx context.Context, code string) (domain.RunRef, error)
	FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error)

	Insert(ctx context.Context, defect domain.Defect) error
	Update(ctx context.Context, defect domain.Defect) error
	FindByCode(ctx context.Context, code shared.Code) (domain.Defect, error)
	List(ctx context.Context, filter Filter) (defects []domain.Defect, total int, err error)
}

// Transition es una transición del flujo del DEF con sus reglas activas.
type Transition struct {
	To             domain.State
	RequiresReason bool
	Rules          []domain.Rule
}

// Workflow responde por los permisos y los estados del flujo del DEF.
type Workflow interface {
	Require(ctx context.Context, roles []string, permission string) error
	InitialState(ctx context.Context) (domain.State, error)
	ResolveTransition(ctx context.Context, from, to string, roles []string) (Transition, error)
}

// Timeline dice cuándo entró cada DEF a su estado actual, a partir de la auditoría.
type Timeline interface {
	// LastTransitions devuelve la fecha por id de DEF; uno que nunca cambió de estado no aparece.
	LastTransitions(ctx context.Context, defectIds []string) (map[string]time.Time, error)
}

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate     AuditAction = "create"
	AuditUpdate     AuditAction = "update"
	AuditTransition AuditAction = "transition"
)

// AuditRecord es un hecho sobre un DEF.
type AuditRecord struct {
	Actor    shared.UserId
	DefectId string
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
