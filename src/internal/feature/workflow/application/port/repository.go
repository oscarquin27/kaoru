// Package port declara lo que necesitan los casos de uso de workflow.
package port

import (
	"context"

	"kaoru/internal/feature/workflow/domain"
)

// Repository lee la configuración de los flujos.
type Repository interface {
	// Grants devuelve los permisos de los roles dados.
	Grants(ctx context.Context, roles []string) ([]domain.Grant, error)
	// Workflow devuelve el flujo de un tipo de entidad con sus reglas.
	Workflow(ctx context.Context, entityType domain.EntityType) (domain.Workflow, error)
}
