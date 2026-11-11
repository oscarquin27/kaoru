// Package port declara lo que necesitan los casos de uso de execution.
package port

import (
	"context"

	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// RunFilter acota la lista de RUN.
type RunFilter struct {
	// FilterTestCases dice si se filtra por TestCaseIds (que puede estar vacío: ningún RUN).
	FilterTestCases bool
	TestCaseIds     []string
	BuildVersion    string
	Page            int
	Size            int
}

// Repository guarda y lee RUN y EXP. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
// No tiene cómo editar ni borrar un RUN.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año para ese prefijo.
	NextCode(ctx context.Context, prefix string, year int) (shared.Code, error)
	FindTestCase(ctx context.Context, code string) (domain.TestCaseRef, error)
	FindRequirement(ctx context.Context, code string) (domain.RequirementRef, error)
	FindUser(ctx context.Context, id shared.UserId) (domain.UserRef, error)

	InsertRun(ctx context.Context, run domain.Run) error
	FindRun(ctx context.Context, code shared.Code) (domain.Run, error)
	ListRuns(ctx context.Context, filter RunFilter) (runs []domain.Run, total int, err error)
	// LatestResults devuelve el resultado del último RUN de cada TC que tenga alguno.
	LatestResults(ctx context.Context, testCaseIds []string) (map[string]domain.Result, error)

	InsertExploration(ctx context.Context, exploration domain.Exploration) error
	UpdateExploration(ctx context.Context, exploration domain.Exploration) error
	FindExploration(ctx context.Context, code shared.Code) (domain.Exploration, error)
	// ListExplorations filtra por código de REQ; vacío no filtra.
	ListExplorations(ctx context.Context, requirementCode string, page, size int) (explorations []domain.Exploration, total int, err error)
}

// TestDesign pregunta al feature testdesign qué TC validan un REQ.
type TestDesign interface {
	// RequirementTestCases devuelve NOT_FOUND_RESOURCE si el REQ no existe.
	RequirementTestCases(ctx context.Context, requirementCode string) ([]domain.CoverageTestCase, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// Tipos de entidad que audita este feature.
const (
	EntityRun         = "test_run"
	EntityExploration = "exploratory_session"
)

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un RUN o un EXP.
type AuditRecord struct {
	Actor      shared.UserId
	EntityType string
	EntityId   string
	Code       shared.Code
	Action     AuditAction
	Field      string
	OldValue   string
	NewValue   string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
