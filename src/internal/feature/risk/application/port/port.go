// Package port declara lo que necesitan los casos de uso de risk.
package port

import (
	"context"

	"kaoru/internal/feature/risk/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository guarda y lee el RISK de cada REQ.
type Repository interface {
	// NextCode reserva el siguiente ID legible del año.
	NextCode(ctx context.Context, year int) (shared.Code, error)
	// FindByRequirement devuelve NOT_FOUND_RESOURCE si el REQ no tiene RISK.
	FindByRequirement(ctx context.Context, requirementId string) (domain.Assessment, error)
	Insert(ctx context.Context, assessment domain.Assessment) error
	Update(ctx context.Context, assessment domain.Assessment) error
}

// RequirementRef es el REQ al que pertenece un RISK.
type RequirementRef struct {
	Id   string
	Code string
}

// Requirements busca REQ en el feature requirement.
type Requirements interface {
	// Find devuelve NOT_FOUND_RESOURCE si el REQ no existe.
	Find(ctx context.Context, code string) (RequirementRef, error)
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

// AuditRecord es un hecho sobre un RISK.
type AuditRecord struct {
	Actor    shared.UserId
	RiskId   string
	Code     shared.Code
	Action   AuditAction
	Field    string
	OldValue string
	NewValue string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
