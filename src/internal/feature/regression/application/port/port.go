// Package port declara lo que necesitan los casos de uso de regression.
package port

import (
	"context"
	"time"

	"kaoru/internal/feature/regression/domain"
	shared "kaoru/internal/shared/domain"
)

// PackageRef es el PKG dueño de la selección.
type PackageRef struct {
	Id        string
	Code      shared.Code
	SystemKey string
}

// ComponentRef es un componente del catálogo.
type ComponentRef struct {
	Code string
	Name string
	Kind string
}

// RunRef es un RUN tal como se muestra junto a un TC.
type RunRef struct {
	Code            string
	Result          string
	BuildVersion    string
	EnvironmentName string
	BuildCode       string
	// EnvironmentCode está vacío si el RUN no se registró sobre un ambiente del catálogo.
	EnvironmentCode string
	ExecutedAt      time.Time
}

// ItemView es un TC de la selección con lo que se muestra de él.
type ItemView struct {
	Item          domain.Item
	Title         string
	IsCritical    bool
	DecidedByName string
	// Components son los componentes del TC que tocan los REQ del PKG.
	Components []ComponentRef
	// LastRun es el último RUN del TC sobre un BUILD del PKG; nil si no tiene.
	LastRun *RunRef
}

// SelectionView es la selección de un PKG con sus TC, por código de TC.
type SelectionView struct {
	Selection       domain.Selection
	ProposedByName  string
	ConfirmedByName string
	Items           []ItemView
}

// Repository guarda la selección de regresión y lee, en el sentido de sus llaves, el PKG, sus REQ,
// sus BUILD, los TC, los componentes y los RUN. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
type Repository interface {
	FindPackage(ctx context.Context, code shared.Code) (PackageRef, error)
	// LockPackage bloquea el PKG hasta el fin de la transacción en curso: toda escritura sobre su
	// selección empieza aquí, exista ya la selección o no.
	LockPackage(ctx context.Context, code shared.Code) (PackageRef, error)
	FindTestCase(ctx context.Context, code shared.Code) (domain.TestCase, error)
	// ListCandidates devuelve los TC activos del sistema del PKG que ejercitan un componente tocado
	// por algún REQ del PKG (impacto) y los críticos; el que cae en los dos es de impacto.
	ListCandidates(ctx context.Context, packageId string) ([]domain.Candidate, error)

	// FindSelection devuelve NOT_FOUND_RESOURCE si el PKG no tiene selección.
	FindSelection(ctx context.Context, packageId string) (SelectionView, error)
	// SaveSelection crea la cabecera o guarda sus cambios.
	SaveSelection(ctx context.Context, selection domain.Selection) error
	// ReplaceItems deja como TC de la selección exactamente los dados.
	ReplaceItems(ctx context.Context, packageId string, items []domain.Item) error
	// SaveItem agrega el TC a la selección o guarda su ajuste.
	SaveItem(ctx context.Context, packageId string, item domain.Item) error
	DeleteItem(ctx context.Context, packageId, testCaseId string) error
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate   AuditAction = "create"
	AuditUpdate   AuditAction = "update"
	AuditDecision AuditAction = "decision"
)

// AuditRecord es un hecho sobre la regresión de un PKG. La entidad auditada es el PKG.
type AuditRecord struct {
	Actor     shared.UserId
	PackageId string
	Code      shared.Code
	Action    AuditAction
	Field     string
	OldValue  string
	NewValue  string
	Reason    string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
