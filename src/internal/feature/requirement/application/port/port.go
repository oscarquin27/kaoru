// Package port declara lo que necesitan los casos de uso de requirement.
package port

import (
	"context"
	"time"

	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// Filter acota la lista de REQ. Un campo vacío no filtra.
type Filter struct {
	SystemKey  string
	StateKey   string
	ChangeCode string
	OwnerId    string
	Query      string
	Page       int
	Size       int
}

// Repository guarda y lee los REQ, sus enlaces y su contrato de entrada.
// Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	FindSystem(ctx context.Context, key string) (domain.System, error)
	FindChange(ctx context.Context, code string) (domain.ChangeRef, error)
	FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error)

	Insert(ctx context.Context, requirement domain.Requirement) error
	Update(ctx context.Context, requirement domain.Requirement) error
	FindByCode(ctx context.Context, code shared.Code) (domain.Requirement, error)
	List(ctx context.Context, filter Filter) (requirements []domain.Requirement, total int, err error)

	InsertLink(ctx context.Context, link domain.Link) error
	UpdateLink(ctx context.Context, link domain.Link) error
	DeleteLink(ctx context.Context, linkId string) error
	FindLink(ctx context.Context, requirementId, linkId string) (domain.Link, error)
	ListLinks(ctx context.Context, requirementId string) ([]domain.Link, error)
	// FindDocumentId devuelve la llave del DOC con ese código.
	FindDocumentId(ctx context.Context, code string) (string, error)

	// ListEntryContract devuelve los ítems activos del contrato con su estado para el REQ.
	ListEntryContract(ctx context.Context, requirementId string) ([]domain.EntryItem, error)
	CheckEntryItem(ctx context.Context, requirementId, itemKey, note string, by shared.UserId, at time.Time) error
	// UncheckEntryItem devuelve NOT_FOUND_RESOURCE si el ítem no estaba marcado.
	UncheckEntryItem(ctx context.Context, requirementId, itemKey string) error
}

// Transition es una transición del flujo del REQ con sus reglas activas.
type Transition struct {
	To             domain.State
	RequiresReason bool
	Rules          []domain.Rule
}

// Workflow responde por los permisos y los estados del flujo del REQ.
type Workflow interface {
	Require(ctx context.Context, roles []string, permission string) error
	InitialState(ctx context.Context) (domain.State, error)
	// ResolveTransition comprueba que el flujo permita el paso y que el rol tenga su permiso.
	ResolveTransition(ctx context.Context, from, to string, roles []string) (Transition, error)
	// Outgoing devuelve las transiciones que salen de un estado, sin mirar permisos.
	Outgoing(ctx context.Context, from string) ([]Transition, error)
}

// Facts trae de otros features los hechos que las reglas de bloqueo necesitan.
// bootstrap conecta sus fuentes después de armar los módulos (decisión D36).
type Facts interface {
	// Risk devuelve nil si el riesgo no está evaluado.
	Risk(ctx context.Context, code shared.Code) (*domain.RiskFact, error)
	Coverage(ctx context.Context, code shared.Code) (domain.CoverageFact, error)
	OpenDefects(ctx context.Context, code shared.Code) ([]domain.DefectFact, error)
}

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate     AuditAction = "create"
	AuditUpdate     AuditAction = "update"
	AuditTransition AuditAction = "transition"
	AuditDelete     AuditAction = "delete"
)

// AuditRecord es un hecho sobre un REQ.
type AuditRecord struct {
	Actor         shared.UserId
	RequirementId string
	Code          shared.Code
	Action        AuditAction
	Field         string
	OldValue      string
	NewValue      string
	Reason        string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
