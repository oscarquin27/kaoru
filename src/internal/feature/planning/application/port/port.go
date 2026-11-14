// Package port declara lo que necesitan los casos de uso de planning.
package port

import (
	"context"
	"time"

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
	// MilestonesBetween devuelve los hitos con fecha entre from y to, ambos incluidos.
	MilestonesBetween(ctx context.Context, from, to time.Time) ([]domain.Milestone, error)

	Activities
}

// ActivityFilter acota la lista de actividades. Vacío no filtra; Today decide el estado.
type ActivityFilter struct {
	Operator   string
	ChangeCode string
	Kind       string
	Status     string
	From       *time.Time
	To         *time.Time
	Today      time.Time
	Page       int
	Size       int
}

// Activities guarda y lee las actividades. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
// Una actividad no se borra: se cancela.
type Activities interface {
	NextActivityCode(ctx context.Context, year int) (shared.Code, error)
	// InsertActivity guarda la actividad con sus operadores.
	InsertActivity(ctx context.Context, activity domain.Activity) error
	// UpdateActivity guarda los datos editables y reemplaza los operadores.
	UpdateActivity(ctx context.Context, activity domain.Activity) error
	// LockActivity bloquea la actividad hasta el fin de la transacción en curso.
	LockActivity(ctx context.Context, code string) error
	FindActivity(ctx context.Context, code string) (domain.Activity, error)
	// ListActivities ordena por inicio, fin y código.
	ListActivities(ctx context.Context, filter ActivityFilter) (activities []domain.Activity, total int, err error)
	// ActivitiesBetween devuelve las no canceladas que tocan los días de from a to.
	ActivitiesBetween(ctx context.Context, from, to time.Time) ([]domain.Activity, error)
	// ActivitiesOfOperators devuelve las no canceladas de esos operadores que tocan los días de from a to.
	ActivitiesOfOperators(ctx context.Context, operators []shared.UserId, from, to time.Time) ([]domain.Activity, error)
	// Certifications devuelve las actividades de certificación de un CHG, también las canceladas.
	Certifications(ctx context.Context, changeCode string) ([]domain.Activity, error)
	// FindChange lee el CHG de una actividad por su código.
	FindChange(ctx context.Context, code string) (domain.ChangeRef, error)
	// FindUsers lee los operadores; los que no existen no vienen.
	FindUsers(ctx context.Context, ids []shared.UserId) ([]domain.UserRef, error)
}

// Changes lee las ventanas de los CHG vigentes.
type Changes interface {
	Windows(ctx context.Context) ([]domain.Window, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// Tipos de entidad que audita este feature.
const (
	EntityMilestone = "milestone"
	EntityActivity  = "activity"
)

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un hito o una actividad. Entity vacío es un hito, que no tiene ID legible.
type AuditRecord struct {
	Actor    shared.UserId
	Entity   string
	EntityId string
	Code     string
	Action   AuditAction
	Field    string
	OldValue string
	NewValue string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
