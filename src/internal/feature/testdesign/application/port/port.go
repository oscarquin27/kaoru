// Package port declara lo que necesitan los casos de uso de testdesign.
package port

import (
	"context"

	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// TestCaseFilter acota la lista de TC. Un campo vacío o nil no filtra.
type TestCaseFilter struct {
	SystemKey     string
	Critical      *bool
	Query         string
	RequirementId string
	Page          int
	Size          int
}

// Repository guarda y lee los UC y los TC. Las búsquedas devuelven NOT_FOUND_RESOURCE si no hay fila.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año para ese prefijo.
	NextCode(ctx context.Context, prefix string, year int) (shared.Code, error)
	// FindSystem devuelve el nombre del sistema.
	FindSystem(ctx context.Context, key string) (name string, err error)

	InsertFunctionalCase(ctx context.Context, functionalCase domain.FunctionalCase) error
	UpdateFunctionalCase(ctx context.Context, functionalCase domain.FunctionalCase) error
	FindFunctionalCase(ctx context.Context, code shared.Code) (domain.FunctionalCase, error)
	ListFunctionalCases(ctx context.Context, requirementId string) ([]domain.FunctionalCase, error)
	// FunctionalCaseRefs devuelve los UC que existen entre los códigos dados.
	FunctionalCaseRefs(ctx context.Context, codes []string) ([]domain.FunctionalCaseRef, error)

	InsertTestCase(ctx context.Context, testCase domain.TestCase) error
	// UpdateTestCase guarda el TC y reemplaza los UC que valida.
	UpdateTestCase(ctx context.Context, testCase domain.TestCase) error
	FindTestCase(ctx context.Context, code shared.Code) (domain.TestCase, error)
	ListTestCases(ctx context.Context, filter TestCaseFilter) (testCases []domain.TestCase, total int, err error)
}

// Requirements busca REQ en el feature requirement.
type Requirements interface {
	// Find devuelve NOT_FOUND_RESOURCE si el REQ no existe.
	Find(ctx context.Context, code string) (domain.RequirementRef, error)
}

// Permissions comprueba los permisos de los roles de un actor.
type Permissions interface {
	Require(ctx context.Context, roles []string, permission string) error
}

// Tipos de entidad que audita este feature.
const (
	EntityFunctionalCase = "functional_case"
	EntityTestCase       = "test_case"
)

// AuditAction es la clase de hecho que se audita.
type AuditAction string

// Acciones que audita este feature.
const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
)

// AuditRecord es un hecho sobre un UC o un TC.
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
