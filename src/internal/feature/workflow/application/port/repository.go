// Package port declara lo que necesitan los casos de uso de workflow.
package port

import (
	"context"
	"time"

	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// Repository lee la configuración de los flujos.
type Repository interface {
	// Grants devuelve los permisos de los roles dados.
	Grants(ctx context.Context, roles []string) ([]domain.Grant, error)
	// Workflow devuelve el flujo de un tipo de entidad con sus reglas.
	Workflow(ctx context.Context, entityType domain.EntityType) (domain.Workflow, error)

	Permissions(ctx context.Context) ([]domain.Permission, error)
	AllGrants(ctx context.Context) ([]domain.Grant, error)
	// ReplaceRoleGrants deja al rol exactamente con esos permisos.
	ReplaceRoleGrants(ctx context.Context, role string, permissions []string) error
	// FindRule devuelve NOT_FOUND_RESOURCE si la regla no existe.
	FindRule(ctx context.Context, id string) (domain.Rule, error)
	UpdateRule(ctx context.Context, rule domain.Rule, now time.Time) error
}

// AuditRecord es un cambio de configuración a auditar.
type AuditRecord struct {
	Actor shared.UserId
	// Entity es "role_permission" o "blocking_rule"; Key identifica el rol o la regla.
	Entity   string
	Key      string
	Field    string
	OldValue string
	NewValue string
}

// AuditTrail registra en la auditoría dentro de la transacción en curso.
// bootstrap lo conecta después de armar audit, que a su vez depende de workflow (decisión D30).
type AuditTrail interface {
	Record(ctx context.Context, record AuditRecord) error
}
